package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
)

type RenderCommand struct {
	*cmds.CommandDescription
}

type RenderSettings struct {
	Layout         string `glazed:"layout"`
	Out            string `glazed:"out"`
	Format         string `glazed:"format"`
	Selector       string `glazed:"selector"`
	Threshold      int    `glazed:"threshold"`
	ViewportWidth  int    `glazed:"viewport-width"`
	ViewportHeight int    `glazed:"viewport-height"`
	WaitMS         int    `glazed:"wait-ms"`
	DebugDir       string `glazed:"debug-dir"`
	WebDir         string `glazed:"web-dir"`
	ChromePath     string `glazed:"chrome-path"`
	ChromeWSURL    string `glazed:"chrome-ws-url"`
}

func newRenderCommand() (*RenderCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cfg := LoadConfig()
	desc := cmds.NewCommandDescription(
		"render",
		cmds.WithShort("Render an Almanach layout once to PNG or bitmap"),
		cmds.WithLong(`Render a JSON/YAML Almanach Studio layout or ZIP layout bundle using Chrome headless.

Examples:
  almanach-render-service render --layout daily.yaml --out daily.png
  almanach-render-service render --layout layout-bundle.zip --out daily.png
  almanach-render-service render --layout daily.yaml --format bitmap --out daily.bin
  almanach-render-service render --layout daily.yaml --selector .paper-shell --debug-dir /tmp/almanach-debug
`),
		cmds.WithFlags(renderFields(cfg)...),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)
	return &RenderCommand{CommandDescription: desc}, nil
}

func renderFields(cfg Config) []*fields.Definition {
	return []*fields.Definition{
		fields.New("layout", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Layout file or ZIP bundle to render. Accepts JSON, YAML, or .zip.")),
		fields.New("out", fields.TypeString, fields.WithHelp("Output artifact path"), fields.WithRequired(true)),
		fields.New("format", fields.TypeChoice, fields.WithDefault("png"), fields.WithChoices("png", "bitmap"), fields.WithHelp("Output format")),
		fields.New("selector", fields.TypeString, fields.WithDefault(".paper-body"), fields.WithHelp("CSS selector to screenshot")),
		fields.New("threshold", fields.TypeInteger, fields.WithDefault(128), fields.WithHelp("Grayscale threshold for bitmap conversion")),
		fields.New("viewport-width", fields.TypeInteger, fields.WithDefault(800), fields.WithHelp("Chrome viewport width")),
		fields.New("viewport-height", fields.TypeInteger, fields.WithDefault(3000), fields.WithHelp("Chrome viewport height")),
		fields.New("wait-ms", fields.TypeInteger, fields.WithDefault(250), fields.WithHelp("Extra wait after loading layout")),
		fields.New("debug-dir", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Directory for debug artifacts")),
		fields.New("web-dir", fields.TypeString, fields.WithDefault(cfg.WebDir), fields.WithHelp("SPA dist directory")),
		fields.New("chrome-path", fields.TypeString, fields.WithDefault(cfg.ChromePath), fields.WithHelp("Chrome/Chromium executable path")),
		fields.New("chrome-ws-url", fields.TypeString, fields.WithDefault(cfg.ChromeWSURL), fields.WithHelp("Remote Chrome websocket URL")),
	}
}

func (c *RenderCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	s := &RenderSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	cfg := LoadConfig()
	layoutSource, err := layoutJSONFromPathOrDefault(s.Layout, cfg)
	if err != nil {
		return err
	}

	opts := renderOptionsFromSettings(s, layoutSource.RenderOptions)
	result, err := renderOneShot(ctx, oneShotRenderRequest{
		LayoutJSON:  layoutSource.LayoutJSON,
		WebDir:      s.WebDir,
		ChromePath:  s.ChromePath,
		ChromeWSURL: s.ChromeWSURL,
		Options:     opts,
	})
	if err != nil {
		return err
	}

	var data []byte
	switch s.Format {
	case "png":
		data = result.PNG
	case "bitmap":
		data = result.Bitmap.Data
	default:
		return fmt.Errorf("unsupported format %q", s.Format)
	}
	if err := os.WriteFile(s.Out, data, 0o600); err != nil {
		return fmt.Errorf("write output %s: %w", s.Out, err)
	}

	return gp.AddRow(ctx, types.NewRow(
		types.MRP("artifact", s.Out),
		types.MRP("format", s.Format),
		types.MRP("selector", result.Selector),
		types.MRP("width", result.Bitmap.Width),
		types.MRP("height", result.Bitmap.Height),
		types.MRP("bytes", len(data)),
		types.MRP("threshold", opts.Threshold),
		types.MRP("rendered_at", result.RenderedAt),
		types.MRP("debug_dir", s.DebugDir),
	))
}

func renderOptionsFromSettings(s *RenderSettings, fileOptions map[string]interface{}) RenderOptions {
	selector := stringFromRenderOptions(fileOptions, "selector", s.Selector)
	threshold := intFromRenderOptions(fileOptions, "threshold", s.Threshold)
	viewportWidth := intFromRenderOptions(fileOptions, "viewportWidth", s.ViewportWidth)
	viewportHeight := intFromRenderOptions(fileOptions, "viewportHeight", s.ViewportHeight)

	return RenderOptions{
		Selector:       selector,
		Threshold:      clampToUint8(threshold),
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
		WaitAfterLoad:  time.Duration(s.WaitMS) * time.Millisecond,
		DebugDir:       s.DebugDir,
		CollectMetrics: s.DebugDir != "",
	}
}

// clampToUint8 clamps val to [0, 255] to prevent integer overflow on int→uint8 conversion.
func clampToUint8(val int) uint8 {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(val)
}
