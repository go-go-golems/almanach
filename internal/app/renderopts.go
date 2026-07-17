package app

import (
	"encoding/json"
	"fmt"

	layoutv1 "github.com/go-go-golems/almanach/gen/almanach/layout/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// renderopts.go replaces the hand-parsed `render:` keys (intFromRenderOptions/
// stringFromRenderOptions) with the typed, validated layout RenderOptions proto
// message (DSL v2, Phase 5). A layout's page-level `render:` object and each
// block's `render:` override are decoded into layoutv1.RenderOptions, validated
// once, and overlaid onto the internal RenderOptions struct.

// parseRenderOptions decodes a `render:` object (as produced by JSON/YAML layout
// parsing into a map) into the typed proto message, tolerating unknown keys.
// Returns (nil, nil) for an empty object so callers keep their defaults.
func parseRenderOptions(m map[string]interface{}) (*layoutv1.RenderOptions, error) {
	if len(m) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal render options: %w", err)
	}
	opts := &layoutv1.RenderOptions{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, opts); err != nil {
		return nil, fmt.Errorf("decode render options: %w", err)
	}
	if err := validateRenderOptions(opts); err != nil {
		return nil, err
	}
	return opts, nil
}

// validateRenderOptions rejects out-of-range values up front instead of letting
// them silently clamp deep in the pipeline.
func validateRenderOptions(o *layoutv1.RenderOptions) error {
	if o.Threshold != nil && *o.Threshold > 255 {
		return fmt.Errorf("render.threshold must be 0-255, got %d", *o.Threshold)
	}
	if o.SupersampleScale != nil && (*o.SupersampleScale < 1 || *o.SupersampleScale > 8) {
		return fmt.Errorf("render.supersampleScale must be 1-8, got %d", *o.SupersampleScale)
	}
	if o.ViewportWidth != nil && *o.ViewportWidth <= 0 {
		return fmt.Errorf("render.viewportWidth must be > 0, got %d", *o.ViewportWidth)
	}
	if o.ViewportHeight != nil && *o.ViewportHeight <= 0 {
		return fmt.Errorf("render.viewportHeight must be > 0, got %d", *o.ViewportHeight)
	}
	// The K118/AtomS3R firmware (firmware/atoms3r/main/web_server.c) accepts
	// density 0..39 and only a discrete set of speeds; reject anything else here
	// so a bad layout fails loudly instead of printing at the previous setting.
	if o.PrinterDensity != nil && (*o.PrinterDensity < 0 || *o.PrinterDensity > 39) {
		return fmt.Errorf("render.printerDensity must be 0-39, got %d", *o.PrinterDensity)
	}
	if o.PrinterSpeed != nil && !supportedPrinterSpeeds[*o.PrinterSpeed] {
		return fmt.Errorf("render.printerSpeed must be one of %v, got %d", printerSpeedList, *o.PrinterSpeed)
	}
	return nil
}

// printerSpeedList mirrors the firmware's web_speed_supported table.
var printerSpeedList = []int32{25, 30, 37, 50, 56, 62, 70, 80, 90, 100, 120, 150, 180, 200, 220}

var supportedPrinterSpeeds = func() map[int32]bool {
	m := make(map[int32]bool, len(printerSpeedList))
	for _, s := range printerSpeedList {
		m[s] = true
	}
	return m
}()

// rasterModeString maps the proto enum to the lowercase name the rasterizer uses.
func rasterModeString(m layoutv1.RasterMode) string {
	switch m {
	case layoutv1.RasterMode_RASTER_MODE_THRESHOLD:
		return "threshold"
	case layoutv1.RasterMode_RASTER_MODE_ATKINSON:
		return "atkinson"
	case layoutv1.RasterMode_RASTER_MODE_FLOYD_STEINBERG:
		return "floydSteinberg"
	case layoutv1.RasterMode_RASTER_MODE_BAYER:
		return "bayer"
	case layoutv1.RasterMode_RASTER_MODE_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}

// applyRenderOptions overlays the set fields of a typed RenderOptions message
// onto a base RenderOptions struct. Unset proto fields (nil / UNSPECIFIED) leave
// the base value untouched, so this composes with CLI defaults and per-block
// overrides on top of page-level options.
func applyRenderOptions(base RenderOptions, p *layoutv1.RenderOptions) RenderOptions {
	if p == nil {
		return base
	}
	if p.Selector != nil && *p.Selector != "" {
		base.Selector = *p.Selector
	}
	if p.Threshold != nil {
		base.Threshold = clampToUint8(int(*p.Threshold))
	}
	if p.SupersampleScale != nil {
		base.SupersampleScale = int(*p.SupersampleScale)
	}
	if p.ViewportWidth != nil {
		base.ViewportWidth = int(*p.ViewportWidth)
	}
	if p.ViewportHeight != nil {
		base.ViewportHeight = int(*p.ViewportHeight)
	}
	if mode := rasterModeString(p.RasterMode); mode != "" {
		base.RasterMode = mode
	}
	if p.Gamma != nil {
		base.Gamma = *p.Gamma
	}
	if p.PrinterDensity != nil {
		base.PrinterDensity = int(*p.PrinterDensity)
	}
	if p.PrinterSpeed != nil {
		base.PrinterSpeed = int(*p.PrinterSpeed)
	}
	return base
}

// perBlockRenderOptions extracts each block's `render:` override from a layout
// JSON document, keyed by block id. Blocks without an id or a render override
// are skipped. Used for block-aware rasterization / per-segment heat (Phase 6).
func perBlockRenderOptions(layoutJSON string) (map[string]*layoutv1.RenderOptions, error) {
	var doc struct {
		Blocks []struct {
			ID     string          `json:"id"`
			Render json.RawMessage `json:"render"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(layoutJSON), &doc); err != nil {
		return nil, fmt.Errorf("parse layout for per-block render: %w", err)
	}
	out := map[string]*layoutv1.RenderOptions{}
	for _, b := range doc.Blocks {
		if b.ID == "" || len(b.Render) == 0 || string(b.Render) == "null" {
			continue
		}
		opts := &layoutv1.RenderOptions{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b.Render, opts); err != nil {
			return nil, fmt.Errorf("decode render for block %q: %w", b.ID, err)
		}
		if err := validateRenderOptions(opts); err != nil {
			return nil, fmt.Errorf("block %q: %w", b.ID, err)
		}
		out[b.ID] = opts
	}
	return out, nil
}
