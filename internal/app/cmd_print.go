package app

import (
	"context"
	"fmt"
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

type PrintCommand struct {
	*cmds.CommandDescription
}

type PrintSettings struct {
	Layout         string  `glazed:"layout"`
	PrinterIP      string  `glazed:"printer-ip"`
	PrinterURL     string  `glazed:"printer-url"`
	FeedLines      int     `glazed:"feed-lines"`
	DryRun         bool    `glazed:"dry-run"`
	Selector       string  `glazed:"selector"`
	Threshold      int     `glazed:"threshold"`
	RasterMode     string  `glazed:"raster-mode"`
	Gamma          float64 `glazed:"gamma"`
	Brightness     float64 `glazed:"brightness"`
	Contrast       float64 `glazed:"contrast"`
	Density        int     `glazed:"density"`
	Speed          int     `glazed:"speed"`
	ViewportWidth  int     `glazed:"viewport-width"`
	ViewportHeight int     `glazed:"viewport-height"`
	WaitMS         int     `glazed:"wait-ms"`
	DebugDir       string  `glazed:"debug-dir"`
	WebDir         string  `glazed:"web-dir"`
	ChromePath     string  `glazed:"chrome-path"`
	ChromeWSURL    string  `glazed:"chrome-ws-url"`
	Data           string  `glazed:"data"`
	Define         string  `glazed:"define"`
}

func newPrintCommand() (*PrintCommand, error) {
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
		"print",
		cmds.WithShort("Render an Almanach layout once and send it to the ESP32 printer"),
		cmds.WithLong(`Render a JSON/YAML layout or ZIP layout bundle, convert it to a 1-bit bitmap, and POST it to stoms3r.

Examples:
  almanach-render-service print --layout daily.yaml --printer-ip 192.168.0.126
  almanach-render-service print --layout layout-bundle.zip --printer-ip 192.168.0.126
  almanach-render-service print --layout daily.yaml --printer-ip 192.168.0.126 --dry-run --output yaml
`),
		cmds.WithFlags(
			fields.New("layout", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Layout file or ZIP bundle to print. Accepts JSON, YAML, or .zip.")),
			fields.New("printer-ip", fields.TypeString, fields.WithDefault(cfg.PrinterIP), fields.WithHelp("ESP32 stoms3r printer IP/host")),
			fields.New("printer-url", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Full printer bitmap endpoint URL; overrides printer-ip")),
			fields.New("feed-lines", fields.TypeInteger, fields.WithDefault(cfg.FeedLines), fields.WithHelp("Feed lines after print")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Render but do not post to printer")),
			fields.New("selector", fields.TypeString, fields.WithDefault(".paper-body"), fields.WithHelp("CSS selector to screenshot")),
			fields.New("threshold", fields.TypeInteger, fields.WithDefault(128), fields.WithHelp("Grayscale threshold for bitmap conversion")),
			fields.New("raster-mode", fields.TypeString, fields.WithDefault("threshold"), fields.WithHelp("1-bit conversion: threshold|atkinson|floyd-steinberg|bayer8. atkinson suits photos")),
			fields.New("gamma", fields.TypeFloat, fields.WithDefault(1.0), fields.WithHelp("Tone-curve exponent for dither modes; <1 lightens midtones (photo default 0.8)")),
			fields.New("brightness", fields.TypeFloat, fields.WithDefault(0.0), fields.WithHelp("Brightness added to normalized gray (dither modes)")),
			fields.New("contrast", fields.TypeFloat, fields.WithDefault(1.0), fields.WithHelp("Contrast scale around 0.5 (dither modes)")),
			fields.New("density", fields.TypeInteger, fields.WithDefault(-1), fields.WithHelp("Printer heat/density 0..39; -1 leaves firmware default")),
			fields.New("speed", fields.TypeInteger, fields.WithDefault(-1), fields.WithHelp("Printer speed; lower = darker; -1 leaves firmware default")),
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
	return &PrintCommand{CommandDescription: desc}, nil
}

func (c *PrintCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	s := &PrintSettings{}
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

	opts := RenderOptions{
		Selector:       stringFromRenderOptions(layoutSource.RenderOptions, "selector", s.Selector),
		Threshold:      clampToUint8(intFromRenderOptions(layoutSource.RenderOptions, "threshold", s.Threshold)),
		RasterMode:     stringFromRenderOptions(layoutSource.RenderOptions, "rasterMode", s.RasterMode),
		Gamma:          floatFromRenderOptions(layoutSource.RenderOptions, "gamma", s.Gamma),
		Brightness:     floatFromRenderOptions(layoutSource.RenderOptions, "brightness", s.Brightness),
		Contrast:       floatFromRenderOptions(layoutSource.RenderOptions, "contrast", s.Contrast),
		ViewportWidth:  intFromRenderOptions(layoutSource.RenderOptions, "viewportWidth", s.ViewportWidth),
		ViewportHeight: intFromRenderOptions(layoutSource.RenderOptions, "viewportHeight", s.ViewportHeight),
		WaitAfterLoad:  time.Duration(s.WaitMS) * time.Millisecond,
		DebugDir:       s.DebugDir,
		CollectMetrics: s.DebugDir != "",
	}

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

	printerURL := s.PrinterURL
	if printerURL == "" && s.PrinterIP != "" {
		printerURL = fmt.Sprintf("http://%s/api/print/bitmap", s.PrinterIP)
	}
	if printerURL == "" && !s.DryRun {
		return fmt.Errorf("printer not configured: set --printer-ip or --printer-url, or use --dry-run")
	}

	printed := false
	printerOK := false
	var printerResponse map[string]any
	if !s.DryRun {
		// Set printer heat (density/speed) before the bitmap. -1 leaves the
		// firmware default. Photo default is a cooler density; text wants hotter.
		var density, speed *int
		if s.Density >= 0 {
			density = &s.Density
		}
		if s.Speed >= 0 {
			speed = &s.Speed
		}
		if density != nil || speed != nil {
			if err = setPrinterHeat(printerURL, density, speed); err != nil {
				return err
			}
		}
		printerResponse, err = sendBitmapToPrinter(printerURL, result.Bitmap, s.FeedLines)
		if err != nil {
			return err
		}
		printed = true
		printerOK = true
	}

	return gp.AddRow(ctx, types.NewRow(
		types.MRP("printed", printed),
		types.MRP("dry_run", s.DryRun),
		types.MRP("printer_url", printerURL),
		types.MRP("width", result.Bitmap.Width),
		types.MRP("height", result.Bitmap.Height),
		types.MRP("bytes", len(result.Bitmap.Data)),
		types.MRP("feed_lines", s.FeedLines),
		types.MRP("selector", result.Selector),
		types.MRP("printer_ok", printerOK),
		types.MRP("printer_response", printerResponse),
		types.MRP("rendered_at", result.RenderedAt),
		types.MRP("debug_dir", s.DebugDir),
	))
}
