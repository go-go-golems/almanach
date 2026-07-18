package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const renderResponseWriteOverhead = 15 * time.Second

// httpWriteTimeout allows a completed render to be serialized and written after
// its Chrome deadline. It must never be shorter than the render deadline.
func httpWriteTimeout(renderTimeout time.Duration) time.Duration {
	if renderTimeout <= 0 {
		renderTimeout = defaultChromeRenderTimeout
	}
	return renderTimeout + renderResponseWriteOverhead
}

type serveSettings struct {
	Port          int
	WebDir        string
	PrinterIP     string
	ChromePath    string
	ChromeWSURL   string
	PaperWidth    int
	BodyScale     float64
	FeedLines     int
	DefaultTheme  string
	LogLevel      string
	StateFile     string
	RenderTimeout time.Duration
}

func serveSettingsFromConfig(cfg Config) serveSettings {
	return serveSettings(cfg)
}

func configFromServeSettings(s serveSettings) Config {
	return Config(s)
}

func NewServeCommand() *cobra.Command {
	defaults := serveSettingsFromConfig(LoadConfig())
	settings := defaults

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Almanach Render Service HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunServe(cmd.Context(), configFromServeSettings(settings))
		},
	}

	cmd.Flags().IntVar(&settings.Port, "port", defaults.Port, "HTTP listen port")
	cmd.Flags().StringVar(&settings.WebDir, "web-dir", defaults.WebDir, "Almanach Studio SPA dist directory")
	cmd.Flags().StringVar(&settings.PrinterIP, "printer-ip", defaults.PrinterIP, "ESP32 stoms3r printer IP/host")
	cmd.Flags().StringVar(&settings.ChromePath, "chrome-path", defaults.ChromePath, "Chrome/Chromium executable path for local mode")
	cmd.Flags().StringVar(&settings.ChromeWSURL, "chrome-ws-url", defaults.ChromeWSURL, "Remote Chrome websocket URL")
	cmd.Flags().IntVar(&settings.PaperWidth, "paper-width", defaults.PaperWidth, "Default paper width in pixels")
	cmd.Flags().Float64Var(&settings.BodyScale, "font-scale", defaults.BodyScale, "Default font/body scale")
	cmd.Flags().IntVar(&settings.FeedLines, "feed-lines", defaults.FeedLines, "Default printer feed lines after print")
	cmd.Flags().StringVar(&settings.DefaultTheme, "default-theme", defaults.DefaultTheme, "Default Almanach theme")
	cmd.Flags().StringVar(&settings.LogLevel, "log-level", defaults.LogLevel, "Log verbosity")
	cmd.Flags().StringVar(&settings.StateFile, "state-file", defaults.StateFile, "Local JSON state file for setup-discovered printer endpoint")

	return cmd
}

func RunServe(ctx context.Context, cfg Config) error {
	return runHTTPServer(ctx, cfg, fmt.Sprintf(":%d", cfg.Port), "Almanach Render Service")
}

func runHTTPServer(ctx context.Context, cfg Config, addr, serviceName string) error {
	log.Printf("%s %s starting on %s", serviceName, Version, addr)
	log.Printf("  Web dir:     %s", cfg.WebDir)
	log.Printf("  Printer IP:  %s", cfg.PrinterIP)
	log.Printf("  Chrome:      %s", cfg.ChromePath)
	log.Printf("  State file:  %s", cfg.StateFile)

	allocatorCtx, allocatorCancel := newChromeAllocator(cfg)
	defer allocatorCancel()

	srv := &Server{
		cfg:           cfg,
		allocatorCtx:  allocatorCtx,
		allocatorDone: allocatorCancel,
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	defer func() { _ = listener.Close() }()
	log.Printf("  Listening:   http://%s", listener.Addr().String())

	httpServer := &http.Server{
		Addr:         listener.Addr().String(),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: httpWriteTimeout(cfg.RenderTimeout),
		IdleTimeout:  120 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(done)

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutting down...")
	case <-done:
		log.Println("Shutting down...")
	case err := <-errCh:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("HTTP shutdown error: %w", err)
	}

	log.Println("Stopped.")
	return nil
}
