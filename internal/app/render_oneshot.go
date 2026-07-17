package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type oneShotRenderRequest struct {
	LayoutJSON  string
	WebDir      string
	ChromePath  string
	ChromeWSURL string
	Options     RenderOptions
}

func renderOneShot(ctx context.Context, req oneShotRenderRequest) (*RenderResult, error) {
	if req.WebDir == "" {
		req.WebDir = LoadConfig().WebDir
	}

	mux := http.NewServeMux()
	registerStaticRoutes(mux, req.WebDir)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen ephemeral render server: %w", err)
	}
	defer func() { _ = ln.Close() }()

	httpServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("ephemeral render server shutdown error: %v", err)
		}
	}()

	cfg := LoadConfig()
	cfg.WebDir = req.WebDir
	cfg.ChromePath = req.ChromePath
	cfg.ChromeWSURL = req.ChromeWSURL

	opts := req.Options.withDefaults()
	if req.Options.RenderTimeout <= 0 {
		opts.RenderTimeout = cfg.RenderTimeout
	}
	opts.BaseURL = "http://" + ln.Addr().String()

	// Per-block render overrides drive block-aware rasterization (Phase 6).
	perBlock, err := perBlockRenderOptions(req.LayoutJSON)
	if err != nil {
		return nil, fmt.Errorf("parse per-block render options: %w", err)
	}
	if len(perBlock) > 0 {
		opts.PerBlockRender = perBlock
	}

	allocatorCtx, allocatorCancel := newChromeAllocatorWithViewport(cfg, opts.ViewportWidth, opts.ViewportHeight, opts.SupersampleScale)
	defer allocatorCancel()

	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("ephemeral render server exited early: %w", err)
		}
		return nil, fmt.Errorf("ephemeral render server exited before render")
	default:
	}

	return renderWithChrome(ctx, allocatorCtx, req.LayoutJSON, opts)
}

func layoutJSONFromObjectOrDefault(obj map[string]interface{}, cfg Config, dataCtx DataContext) (string, map[string]interface{}, error) {
	if len(obj) == 0 {
		layout := buildScaffoldLayout(cfg)
		b, err := json.Marshal(layout)
		if err != nil {
			return "", nil, fmt.Errorf("marshal scaffold: %w", err)
		}
		return string(b), nil, nil
	}

	// Merge data from wrapped request body into dataCtx.
	wrappedDataCtx := dataCtx
	if wrappedData, ok := obj["data"]; ok {
		if dm, ok := wrappedData.(map[string]interface{}); ok {
			if wrappedDataCtx == nil {
				wrappedDataCtx = DataContext{}
			}
			for k, v := range dm {
				wrappedDataCtx[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	renderOptions := map[string]interface{}{}
	layoutObj := any(obj)
	if wrappedLayout, ok := obj["layout"]; ok {
		// Wrapped form: { layout: {...}, render: {...} }.
		layoutObj = wrappedLayout
		if ro, ok := obj["render"].(map[string]interface{}); ok {
			renderOptions = ro
		}
	} else if ro, ok := obj["render"].(map[string]interface{}); ok {
		// Flat form: top-level `render:` alongside `blocks:`. Extract it and drop
		// it from the layout sent to the studio (which ignores it anyway).
		renderOptions = ro
		delete(obj, "render")
	}

	if layoutMap, ok := layoutObj.(map[string]interface{}); ok {
		blocks, ok := layoutMap["blocks"]
		if !ok {
			return "", nil, fmt.Errorf("layout object must contain a blocks array")
		}
		if _, ok := blocks.([]interface{}); !ok {
			return "", nil, fmt.Errorf("layout.blocks must be an array")
		}

		// Resolve template expressions before marshaling.
		if err := ResolveTemplate(layoutMap, wrappedDataCtx); err != nil {
			return "", nil, fmt.Errorf("resolve template: %w", err)
		}
	}

	b, err := json.Marshal(layoutObj)
	if err != nil {
		return "", nil, fmt.Errorf("marshal layout object: %w", err)
	}
	return string(b), renderOptions, nil
}

// The page-level `render:` block is now parsed via the typed proto
// RenderOptions (see renderopts.go); the old intFromRenderOptions /
// stringFromRenderOptions helpers were removed in DSL v2 Phase 5.
