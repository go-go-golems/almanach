package app

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
)

type InspectCommand struct {
	*cmds.CommandDescription
}

type InspectSettings struct {
	Layout         string `glazed:"layout"`
	Selector       string `glazed:"selector"`
	ViewportWidth  int    `glazed:"viewport-width"`
	ViewportHeight int    `glazed:"viewport-height"`
	WaitMS         int    `glazed:"wait-ms"`
	DebugDir       string `glazed:"debug-dir"`
	WebDir         string `glazed:"web-dir"`
	ChromePath     string `glazed:"chrome-path"`
	ChromeWSURL    string `glazed:"chrome-ws-url"`
	Data           string `glazed:"data"`
	Define         string `glazed:"define"`
}

func newInspectCommand() (*InspectCommand, error) {
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
		"inspect",
		cmds.WithShort("Inspect Almanach render DOM metrics for cutoff debugging"),
		cmds.WithLong(`Render a JSON/YAML layout or ZIP layout bundle in Chrome headless and emit DOM metrics for key paper/clipping selectors.

Examples:
  almanach-render-service inspect --layout daily.yaml --output yaml
  almanach-render-service inspect --layout layout-bundle.zip --output yaml
  almanach-render-service inspect --layout daily.yaml --debug-dir /tmp/almanach-debug
`),
		cmds.WithFlags(
			fields.New("layout", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Layout file or ZIP bundle to inspect. Accepts JSON, YAML, or .zip.")),
			fields.New("selector", fields.TypeString, fields.WithDefault(".paper-body"), fields.WithHelp("Primary CSS selector to screenshot/validate")),
			fields.New("viewport-width", fields.TypeInteger, fields.WithDefault(800), fields.WithHelp("Chrome viewport width")),
			fields.New("viewport-height", fields.TypeInteger, fields.WithDefault(3000), fields.WithHelp("Chrome viewport height")),
			fields.New("wait-ms", fields.TypeInteger, fields.WithDefault(250), fields.WithHelp("Extra wait after loading layout")),
			fields.New("debug-dir", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Directory for debug artifacts")),
			fields.New("web-dir", fields.TypeString, fields.WithDefault(cfg.WebDir), fields.WithHelp("SPA dist directory")),
			fields.New("chrome-path", fields.TypeString, fields.WithDefault(cfg.ChromePath), fields.WithHelp("Chrome/Chromium executable path")),
			fields.New("chrome-ws-url", fields.TypeString, fields.WithDefault(cfg.ChromeWSURL), fields.WithHelp("Remote Chrome websocket URL")),
			fields.New("data", fields.TypeString, fields.WithDefault(""), fields.WithHelp("YAML/JSON data context file for template resolution")),
			fields.New("define", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Inline key=value for template variables (comma-separated: -D key=val,key2=val2)")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)
	return &InspectCommand{CommandDescription: desc}, nil
}

func (c *InspectCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	s := &InspectSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	cfg := LoadConfig()

	dataCtx, err := loadDataCtxFromFlags(s.Data, s.Define)
	if err != nil {
		return err
	}

	layoutSource, err := layoutJSONFromPathOrDefault(s.Layout, cfg, dataCtx)
	if err != nil {
		return err
	}

	renderSettings := &RenderSettings{
		Selector:       s.Selector,
		Threshold:      128,
		ViewportWidth:  s.ViewportWidth,
		ViewportHeight: s.ViewportHeight,
		WaitMS:         s.WaitMS,
		DebugDir:       s.DebugDir,
	}
	opts, err := renderOptionsFromSettings(renderSettings, layoutSource.RenderOptions)
	if err != nil {
		return err
	}
	opts.CollectMetrics = true

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

	selectors := []string{".paper-shell", ".paper-body", ".canvas", ".workspace", ".almanach-app"}
	for _, selector := range selectors {
		m := result.Metrics[selector]
		if m == nil {
			if err := gp.AddRow(ctx, types.NewRow(types.MRP("selector", selector), types.MRP("found", false))); err != nil {
				return err
			}
			continue
		}
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("selector", selector),
			types.MRP("found", true),
			types.MRP("x", m.X),
			types.MRP("y", m.Y),
			types.MRP("width", m.Width),
			types.MRP("height", m.Height),
			types.MRP("scroll_width", m.ScrollWidth),
			types.MRP("scroll_height", m.ScrollHeight),
			types.MRP("overflow", m.Overflow),
			types.MRP("overflow_x", m.OverflowX),
			types.MRP("overflow_y", m.OverflowY),
			types.MRP("display", m.Display),
			types.MRP("position", m.Position),
			types.MRP("screenshot_selector", result.Selector),
		)); err != nil {
			return err
		}
	}

	return nil
}
