package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	layoutv1 "github.com/go-go-golems/almanach/gen/almanach/layout/v1"
	"gopkg.in/yaml.v3"
)

const (
	defaultRenderSelector       = ".paper-shell"
	defaultRenderThreshold      = 128
	defaultRenderViewportWidth  = 1200
	defaultRenderViewportHeight = 2000
	defaultRenderWait           = 250 * time.Millisecond
	defaultChromeRenderTimeout  = 2 * time.Minute
)

// RenderOptions controls one Chrome render pass. HTTP handlers use the legacy
// defaults for compatibility; CLI commands can override these fields for
// one-shot preview/print/debug workflows.
type RenderOptions struct {
	BaseURL          string
	Selector         string
	Threshold        uint8
	ThresholdSet     bool // distinguishes an explicit zero cutoff from unset
	SupersampleScale int  // render oversampling factor; downscaled before 1-bit conversion
	ViewportWidth    int
	ViewportHeight   int
	WaitAfterLoad    time.Duration
	RenderTimeout    time.Duration
	DebugDir         string
	CollectMetrics   bool

	// Rasterization + printer knobs from the typed layout RenderOptions
	// (DSL v2). Zero values mean "unset / use default". RasterMode and Gamma
	// feed the block-aware rasterizer (Phase 6); PrinterDensity/PrinterSpeed
	// are applied to the printer before a print.
	RasterMode        string  // "", "threshold", "atkinson", "floydSteinberg", "bayer"
	Gamma             float64 // 0 = unset
	PrinterDensity    int     // valid range 0-39 when PrinterDensitySet
	PrinterDensitySet bool    // distinguishes an explicit firmware minimum (0) from unset
	PrinterSpeed      int     // 0 = unset

	// PerBlockRender maps block id -> its render override, used for block-aware
	// rasterization (Phase 6): a block's bounding box becomes a raster region
	// with the block's mode/gamma/threshold.
	PerBlockRender map[string]*layoutv1.RenderOptions
}

// RenderMetrics contains browser layout measurements for important elements.
type RenderMetrics map[string]*ElementMetrics

// ElementMetrics mirrors the JSON object returned by collectMetricsJS.
type ElementMetrics struct {
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	ScrollWidth  int     `json:"scrollWidth"`
	ScrollHeight int     `json:"scrollHeight"`
	Overflow     string  `json:"overflow"`
	OverflowX    string  `json:"overflowX"`
	OverflowY    string  `json:"overflowY"`
	Display      string  `json:"display"`
	Position     string  `json:"position"`
}

// RenderResult holds the output of a render pass.
// ChromeRenderError identifies the failing browser stage and preserves the
// browser errors observed before the render stopped. HTTP handlers expose these
// fields so clients can distinguish a bad layout from a Chrome timeout.
type ChromeRenderError struct {
	Stage       string
	Elapsed     time.Duration
	Timeout     time.Duration
	Cause       error
	Diagnostics []string
}

func (e *ChromeRenderError) Error() string {
	message := fmt.Sprintf("chrome render stage %q failed after %s (timeout %s): %v", e.Stage, e.Elapsed.Round(time.Millisecond), e.Timeout, e.Cause)
	if len(e.Diagnostics) > 0 {
		message += "; browser: " + strings.Join(e.Diagnostics, " | ")
	}
	return message
}

func (e *ChromeRenderError) Unwrap() error { return e.Cause }

type browserDiagnosticCollector struct {
	mu       sync.Mutex
	messages []string
}

func (c *browserDiagnosticCollector) add(message string) {
	const maxMessages = 8
	const maxMessageLength = 1000
	if len(message) > maxMessageLength {
		message = message[:maxMessageLength] + "…"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.messages) < maxMessages {
		c.messages = append(c.messages, message)
	}
}

func (c *browserDiagnosticCollector) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.messages...)
}

type RenderResult struct {
	Bitmap     *Bitmap
	PNG        []byte // optional PNG for debug/download
	Theme      string
	RenderedAt string
	LayoutJSON string
	Metrics    RenderMetrics
	Selector   string
	Options    RenderOptions

	// HeatRegions are row bands that request a specific printer density, derived
	// from per-block printerDensity render options + block bounding boxes. Used
	// for per-segment heat (Phase 6): text hotter, photos cooler in one page.
	HeatRegions []HeatRegion
}

// HeatRegion is a [YStart, YEnd) band of bitmap rows to print at Density.
type HeatRegion struct {
	YStart  int
	YEnd    int
	Density int
}

// Bitmap represents a 1-bit monochrome image in MSB-first packed format.
type Bitmap struct {
	Width       int
	Height      int
	BytesPerRow int
	Data        []byte
}

// renderLayoutJSON normalizes a request, applies its typed page/per-block render
// options, and renders it through Chrome. HTTP and CLI callers therefore share
// the same layout contract instead of silently falling back to threshold mode.
func (s *Server) render(ctx context.Context, layoutOverride io.Reader) (*RenderResult, error) {
	layoutJSON, rawPageRender, err := s.layoutJSONFromReader(layoutOverride)
	if err != nil {
		return nil, err
	}

	pageRender, err := parseRenderOptions(rawPageRender)
	if err != nil {
		return nil, fmt.Errorf("parse page render options: %w", err)
	}
	perBlock, err := perBlockRenderOptions(layoutJSON)
	if err != nil {
		return nil, fmt.Errorf("parse per-block render options: %w", err)
	}

	opts := applyRenderOptions(defaultHTTPRenderOptions(s.cfg), pageRender)
	opts.PerBlockRender = perBlock
	return renderWithChrome(ctx, s.allocatorCtx, layoutJSON, opts)
}

func (s *Server) layoutJSONFromReader(layoutOverride io.Reader) (string, map[string]interface{}, error) {
	if layoutOverride != nil {
		data, err := io.ReadAll(layoutOverride)
		if err != nil {
			return "", nil, fmt.Errorf("read layout: %w", err)
		}
		if len(bytes.TrimSpace(data)) > 0 {
			return s.layoutJSONFromRaw(data)
		}
	}

	layout := buildScaffoldLayout(s.cfg)
	b, err := json.Marshal(layout)
	if err != nil {
		return "", nil, fmt.Errorf("marshal scaffold: %w", err)
	}
	return string(b), nil, nil
}

// layoutJSONFromRaw parses raw JSON/YAML, extracts a data context and page
// render options, resolves template expressions, and returns normalized layout
// JSON plus the render settings that were intentionally removed from the studio
// payload.
func (s *Server) layoutJSONFromRaw(raw []byte) (string, map[string]interface{}, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		// Not valid JSON — try YAML.
		if yamlErr := yaml.Unmarshal(raw, &obj); yamlErr != nil {
			return "", nil, fmt.Errorf("parse layout: json: %v, yaml: %v", err, yamlErr)
		}
	}
	if len(obj) == 0 {
		return string(raw), nil, nil
	}

	layoutJSON, renderOptions, err := layoutJSONFromObjectOrDefault(obj, s.cfg, nil)
	if err != nil {
		return "", nil, err
	}
	return layoutJSON, renderOptions, nil
}

func defaultHTTPRenderOptions(cfg Config) RenderOptions {
	opts := defaultRenderOptions(fmt.Sprintf("http://localhost:%d", cfg.Port))
	opts.RenderTimeout = cfg.RenderTimeout
	return opts
}

func defaultRenderOptions(baseURL string) RenderOptions {
	return RenderOptions{
		BaseURL:        baseURL,
		Selector:       defaultRenderSelector,
		Threshold:      defaultRenderThreshold,
		ViewportWidth:  defaultRenderViewportWidth,
		ViewportHeight: defaultRenderViewportHeight,
		WaitAfterLoad:  defaultRenderWait,
		CollectMetrics: true,
	}
}

func (o RenderOptions) withDefaults() RenderOptions {
	if o.Selector == "" {
		o.Selector = defaultRenderSelector
	}
	if o.Threshold == 0 && !o.ThresholdSet {
		o.Threshold = defaultRenderThreshold
	}
	if o.ViewportWidth <= 0 {
		o.ViewportWidth = defaultRenderViewportWidth
	}
	if o.ViewportHeight <= 0 {
		o.ViewportHeight = defaultRenderViewportHeight
	}
	if o.WaitAfterLoad <= 0 {
		o.WaitAfterLoad = defaultRenderWait
	}
	if o.RenderTimeout <= 0 {
		o.RenderTimeout = defaultChromeRenderTimeout
	}
	if o.SupersampleScale <= 0 {
		o.SupersampleScale = defaultSupersampleScale
	}
	if o.DebugDir != "" {
		o.CollectMetrics = true
	}
	return o
}

func renderWithChrome(ctx context.Context, allocatorCtx context.Context, layoutJSON string, opts RenderOptions) (*RenderResult, error) {
	start := time.Now()
	opts = opts.withDefaults()
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("render base URL is required")
	}

	log.Printf("[render] Starting Chrome render for %d bytes of layout JSON (selector=%s)", len(layoutJSON), opts.Selector)

	tabCtx, tabCancel := chromedp.NewContext(allocatorCtx)
	defer tabCancel()
	browserDiagnostics := &browserDiagnosticCollector{}
	chromedp.ListenTarget(tabCtx, func(event any) {
		switch e := event.(type) {
		case *runtime.EventExceptionThrown:
			message := "exception: " + e.ExceptionDetails.Error()
			browserDiagnostics.add(message)
			log.Printf("[render] browser %s", message)
		case *runtime.EventConsoleAPICalled:
			if e.Type == runtime.APITypeError || e.Type == runtime.APITypeWarning {
				message := fmt.Sprintf("console %s: %s", e.Type, formatBrowserConsoleArgs(e.Args))
				browserDiagnostics.add(message)
				log.Printf("[render] browser %s", message)
			}
		}
	})

	renderCtx, renderCancel := context.WithTimeout(tabCtx, opts.RenderTimeout)
	defer renderCancel()

	var screenshotBuf []byte
	var metrics RenderMetrics

	layoutArg, err := json.Marshal(layoutJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal layout argument: %w", err)
	}
	selectorArg, err := json.Marshal(opts.Selector)
	if err != nil {
		return nil, fmt.Errorf("marshal screenshot selector: %w", err)
	}

	captureJS, err := injectCaptureCSSJS(captureCSS())
	if err != nil {
		return nil, err
	}

	type renderStage struct {
		name    string
		actions []chromedp.Action
	}
	stages := []renderStage{
		{"viewport", []chromedp.Action{chromedp.EmulateViewport(int64(opts.ViewportWidth), int64(opts.ViewportHeight), chromedp.EmulateScale(float64(opts.SupersampleScale)))}},
		{"navigate", []chromedp.Action{chromedp.Navigate(opts.BaseURL + "/almanach")}},
		{"body-ready", []chromedp.Action{chromedp.WaitVisible("body", chromedp.ByQuery)}},
		{"almanach-ready", []chromedp.Action{chromedp.Poll(`window.almanachReady === true`, nil, chromedp.WithPollingTimeout(10*time.Second))}},
		{"load-layout", []chromedp.Action{chromedp.Evaluate(fmt.Sprintf(`window.almanachLoadLayout(JSON.parse(%s))`, layoutArg), nil)}},
		{"layout-visible", []chromedp.Action{chromedp.Poll(fmt.Sprintf(`document.querySelector(%s) !== null`, selectorArg), nil, chromedp.WithPollingTimeout(10*time.Second))}},
		{"fonts-and-frames", []chromedp.Action{chromedp.Evaluate(waitForFontsAndFramesJS(), nil)}},
		{"settle", []chromedp.Action{chromedp.Sleep(opts.WaitAfterLoad)}},
		{"capture-css", []chromedp.Action{chromedp.Evaluate(captureJS, nil)}},
		{"post-capture-fonts-and-frames", []chromedp.Action{chromedp.Evaluate(waitForFontsAndFramesJS(), nil)}},
	}
	if opts.CollectMetrics {
		stages = append(stages, renderStage{"page-metrics", []chromedp.Action{chromedp.Evaluate(collectMetricsJS(), &metrics)}})
	}
	// Collect per-block bounding boxes when any block carries a render override,
	// so we can rasterize its region differently (Phase 6).
	var blockMetrics []blockMetric
	if len(opts.PerBlockRender) > 0 {
		stages = append(stages, renderStage{"block-metrics", []chromedp.Action{chromedp.Evaluate(collectBlockMetricsJS(opts.Selector), &blockMetrics)}})
	}
	stages = append(stages, renderStage{"screenshot", []chromedp.Action{chromedp.Screenshot(opts.Selector, &screenshotBuf, chromedp.ByQuery, chromedp.NodeVisible)}})

	for _, stage := range stages {
		stageStart := time.Now()
		if err := chromedp.Run(renderCtx, stage.actions...); err != nil {
			renderErr := &ChromeRenderError{
				Stage:       stage.name,
				Elapsed:     time.Since(start),
				Timeout:     opts.RenderTimeout,
				Cause:       err,
				Diagnostics: browserDiagnostics.snapshot(),
			}
			log.Printf("[render] %v", renderErr)
			if opts.DebugDir != "" {
				if debugErr := writeRenderFailureDebugArtifacts(opts.DebugDir, layoutJSON, renderErr); debugErr != nil {
					log.Printf("[render] write failure debug artifacts: %v", debugErr)
				}
			}
			return nil, renderErr
		}
		log.Printf("[render] Chrome stage=%s completed in %s", stage.name, time.Since(stageStart))
		if stage.name == "page-metrics" {
			metricsJSON, _ := json.Marshal(metrics)
			log.Printf("[render] page metrics: %s", metricsJSON)
		}
	}

	log.Printf("[render] Screenshot captured: %d bytes PNG", len(screenshotBuf))

	// Downscale the oversampled screenshot back to the target resolution, then
	// convert to 1-bit. At scale 1 with no per-block regions this is the plain
	// threshold path; per-block render overrides add raster regions (Phase 6).
	// Page-level rasterMode/gamma (Layout.render) cover every row not claimed
	// by a block override, so a full-page dither needs no per-block config.
	regions := blockRasterRegions(blockMetrics, opts.PerBlockRender)
	regions = append(regions, pageRasterRegions(opts.RasterMode, opts.Gamma, regions)...)
	bitmap, err := pngToBitmapSupersampledRegions(screenshotBuf, opts.Threshold, opts.SupersampleScale, regions)
	if err != nil {
		return nil, fmt.Errorf("bitmap convert: %w", err)
	}

	result := &RenderResult{
		Bitmap:      bitmap,
		PNG:         screenshotBuf,
		Theme:       extractThemeFromLayout(layoutJSON),
		RenderedAt:  time.Now().UTC().Format(time.RFC3339),
		LayoutJSON:  layoutJSON,
		Metrics:     metrics,
		Selector:    opts.Selector,
		Options:     opts,
		HeatRegions: blockHeatRegions(blockMetrics, opts.PerBlockRender),
	}

	if opts.DebugDir != "" {
		if err := writeRenderDebugArtifacts(opts.DebugDir, result); err != nil {
			return nil, err
		}
	}

	elapsed := time.Since(start)
	log.Printf("Rendered %dx%d bitmap (%d bytes) in %v", bitmap.Width, bitmap.Height, len(bitmap.Data), elapsed)

	return result, nil
}

func formatBrowserConsoleArgs(args []*runtime.RemoteObject) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == nil {
			continue
		}
		if arg.Description != "" {
			parts = append(parts, arg.Description)
			continue
		}
		if arg.Value != nil {
			parts = append(parts, arg.Value.String())
			continue
		}
		parts = append(parts, string(arg.Type))
	}
	return strings.Join(parts, " ")
}

func injectCaptureCSSJS(css string) (string, error) {
	cssJSON, err := json.Marshal(css)
	if err != nil {
		return "", fmt.Errorf("marshal capture CSS: %w", err)
	}
	return fmt.Sprintf(`(function() {
	var old = document.getElementById('__render-capture');
	if (old) old.remove();
	var s = document.createElement('style');
	s.id = '__render-capture';
	s.textContent = %s;
	document.head.appendChild(s);
	document.querySelectorAll('.block-wrap').forEach(function(el) {
		el.classList.remove('selected');
	});
})();`, string(cssJSON)), nil
}

func captureCSS() string {
	return `
html, body, #root {
  margin: 0 !important;
  padding: 0 !important;
  width: fit-content !important;
  height: auto !important;
  min-height: 0 !important;
  overflow: visible !important;
  background: #ffffff !important;
}
.almanach-app {
  background: #ffffff !important;
  height: auto !important;
  min-height: 0 !important;
  overflow: visible !important;
  display: block !important;
}
.almanach-app::before { display: none !important; }
.topbar, .rail, .block-controls { display: none !important; }
.workspace {
  display: block !important;
  grid-template-columns: none !important;
  height: auto !important;
  min-height: 0 !important;
  overflow: visible !important;
}
.canvas {
  display: block !important;
  padding: 0 !important;
  margin: 0 !important;
  background: #ffffff !important;
  background-image: none !important;
  width: fit-content !important;
  height: auto !important;
  min-height: 0 !important;
  overflow: visible !important;
}
.paper-shell {
  filter: none !important;
  margin: 0 !important;
  box-shadow: none !important;
}
.block-wrap {
  padding: 4px 0 !important;
  cursor: default !important;
  outline: none !important;
}
.block-wrap::before { display: none !important; }
* { -webkit-print-color-adjust: exact !important; print-color-adjust: exact !important; }
`
}

func waitForFontsAndFramesJS() string {
	return `(async function() {
	if (document.fonts && document.fonts.ready) await document.fonts.ready;
	const images = Array.from(document.querySelectorAll('img'));
	await Promise.all(images.map(function(img) {
		if (img.complete && img.naturalWidth > 0) return Promise.resolve();
		return new Promise(function(resolve) {
			img.addEventListener('load', resolve, { once: true });
			img.addEventListener('error', resolve, { once: true });
			setTimeout(resolve, 10000);
		});
	}));
	await new Promise(function(resolve) { requestAnimationFrame(function() { requestAnimationFrame(resolve); }); });
})();`
}

func collectMetricsJS() string {
	return `(() => {
	const names = ['.paper-shell', '.paper-body', '.canvas', '.workspace', '.almanach-app'];
	return Object.fromEntries(names.map((sel) => {
		const el = document.querySelector(sel);
		if (!el) return [sel, null];
		const r = el.getBoundingClientRect();
		const cs = getComputedStyle(el);
		return [sel, {
			x: r.x,
			y: r.y,
			width: r.width,
			height: r.height,
			scrollWidth: el.scrollWidth,
			scrollHeight: el.scrollHeight,
			overflow: cs.overflow,
			overflowX: cs.overflowX,
			overflowY: cs.overflowY,
			display: cs.display,
			position: cs.position
		}];
	}));
})()`
}

func writeRenderFailureDebugArtifacts(debugDir, layoutJSON string, renderErr *ChromeRenderError) error {
	if err := os.MkdirAll(debugDir, 0o750); err != nil {
		return fmt.Errorf("create debug dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(debugDir, "layout.json"), []byte(layoutJSON), 0o600); err != nil {
		return fmt.Errorf("write layout: %w", err)
	}
	diagnostics := strings.Join(renderErr.Diagnostics, "\n\n")
	contents := fmt.Sprintf("stage: %s\nelapsed: %s\ntimeout: %s\nerror: %v\n\nbrowser diagnostics:\n%s\n", renderErr.Stage, renderErr.Elapsed, renderErr.Timeout, renderErr.Cause, diagnostics)
	if err := os.WriteFile(filepath.Join(debugDir, "render-error.txt"), []byte(contents), 0o600); err != nil {
		return fmt.Errorf("write render error: %w", err)
	}
	return nil
}

func writeRenderDebugArtifacts(debugDir string, result *RenderResult) error {
	if err := os.MkdirAll(debugDir, 0o750); err != nil {
		return fmt.Errorf("create debug dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(debugDir, "screenshot.png"), result.PNG, 0o600); err != nil {
		return fmt.Errorf("write debug screenshot: %w", err)
	}
	if err := os.WriteFile(filepath.Join(debugDir, "bitmap.bin"), result.Bitmap.Data, 0o600); err != nil {
		return fmt.Errorf("write debug bitmap: %w", err)
	}
	if err := writePrettyJSON(filepath.Join(debugDir, "layout.json"), json.RawMessage(result.LayoutJSON)); err != nil {
		return err
	}
	if result.Metrics != nil {
		if err := writePrettyJSON(filepath.Join(debugDir, "metrics.json"), result.Metrics); err != nil {
			return err
		}
	}
	return nil
}

func writePrettyJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// newChromeAllocator creates a Chrome allocator. Two modes:
//
//   - If CHROME_WS_URL is set (e.g. "ws://chrome:9222"), connects to a remote
//     headless-shell container. This is the Docker/production mode.
//   - Otherwise, launches a local Chrome process. This is the dev mode.
func newChromeAllocator(cfg Config) (context.Context, context.CancelFunc) {
	return newChromeAllocatorWithViewport(cfg, defaultRenderViewportWidth, defaultRenderViewportHeight, defaultSupersampleScale)
}

func newChromeAllocatorWithViewport(cfg Config, viewportWidth, viewportHeight, scale int) (context.Context, context.CancelFunc) {
	if cfg.ChromeWSURL != "" {
		log.Printf("Chrome mode: remote (%s)", cfg.ChromeWSURL)
		allocCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), cfg.ChromeWSURL)
		return allocCtx, cancel
	}

	if viewportWidth <= 0 {
		viewportWidth = defaultRenderViewportWidth
	}
	if viewportHeight <= 0 {
		viewportHeight = defaultRenderViewportHeight
	}
	if scale <= 0 {
		scale = defaultSupersampleScale
	}

	log.Printf("Chrome mode: local (launching Chrome, supersample=%dx)", scale)
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("force-device-scale-factor", fmt.Sprintf("%d", scale)),
		chromedp.WindowSize(viewportWidth*scale, viewportHeight*scale),
	}

	// At 1x, disable font anti-aliasing so text rasterizes monochrome and small
	// strokes survive the 1-bit conversion. When supersampling (scale>1) we keep
	// AA on: the oversampled render captures thin strokes, and the box-average
	// downscale + threshold reconstructs them crisply while preserving the font's
	// shape — including small serif italics that AA-off at 1x renders roughly
	// (ALMANACH-PIXELFONT).
	if scale <= 1 {
		if env := renderFontEnv(); len(env) > 0 {
			opts = append(opts, chromedp.Env(env...))
		}
	}

	if cfg.ChromePath != "" {
		opts = append(opts, chromedp.ExecPath(cfg.ChromePath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	return allocCtx, cancel
}

func extractThemeFromLayout(layoutJSON string) string {
	var layout struct {
		Theme string `json:"theme"`
	}
	if json.Unmarshal([]byte(layoutJSON), &layout) == nil && layout.Theme != "" {
		return layout.Theme
	}
	return "minimal"
}
