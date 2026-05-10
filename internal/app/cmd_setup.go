package app

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

type setupSettings struct {
	Port   int
	WebDir string
}

func NewSetupCommand() *cobra.Command {
	defaults := LoadConfig()
	settings := setupSettings{
		Port:   defaults.Port,
		WebDir: defaults.WebDir,
	}

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Serve the Almanach BLE printer setup page on localhost",
		Long: "Serve the standalone Almanach BLE printer setup page from local or embedded web assets. " +
			"The setup server binds to 127.0.0.1 so Web Bluetooth can run from localhost without exposing the page on the LAN.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig()
			cfg.Port = settings.Port
			cfg.WebDir = settings.WebDir
			return RunSetup(cmd.Context(), cfg)
		},
	}

	cmd.Flags().IntVar(&settings.Port, "port", defaults.Port, "localhost HTTP listen port for the setup page")
	cmd.Flags().StringVar(&settings.WebDir, "web-dir", defaults.WebDir, "Almanach web dist directory containing setup.html and setup-bundle.js")

	return cmd
}

func RunSetup(ctx context.Context, cfg Config) error {
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	log.Printf("Open setup page: http://localhost:%d/setup", cfg.Port)
	return runHTTPServer(ctx, cfg, addr, "Almanach Setup Server")
}
