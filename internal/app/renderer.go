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
	"time"

	"github.com/chromedp/chromedp"
	"gopkg.in/yaml.v3"
)

const (
	defaultRenderSelector       = ".paper-shell"
	defaultRenderThreshold      = 128
	defaultRenderViewportWidth  = 1200
	defaultRenderViewportHeight = 2000
	defaultRenderWait           = 250 * time.Millisecond
)

// RenderOptions controls one Chrome render pass. HTTP handlers use the legacy
// defaults for compatibility; CLI commands can override these fields for
// one-shot preview/print/debug workflows.
type RenderOptions struct {
	BaseURL        string
	Selector       string
	Threshold      uint8
	ViewportWidth  int
	ViewportHeight int
	WaitAfterLoad  time.Duration
	DebugDir       string
	CollectMetrics bool
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
type RenderResult struct {
	Bitmap     *Bitmap
	PNG        []byte // optional PNG for debug/download
	Theme      string
	RenderedAt string
	LayoutJSON string
	Metrics    RenderMetrics
	Selector   string
}

// Bitmap represents a 1-bit monochrome image in MSB-first packed format.
type Bitmap struct {
	Width       int
	Height      int
	BytesPerRow int
	Data        []byte
}

// renderLayoutJSON builds a layout from fetchers and renders it via Chrome headless.
func (s *Server) render(ctx context.Context, layoutOverride io.Reader) (*RenderResult, error) {
	layoutJSON, err := s.layoutJSONFromReader(layoutOverride)
	if err != nil {
		return nil, err
	}

	return s.renderWithChrome(ctx, layoutJSON)
}

func (s *Server) layoutJSONFromReader(layoutOverride io.Reader) (string, error) {
	if layoutOverride != nil {
		data, err := io.ReadAll(layoutOverride)
		if err != nil {
			return "", fmt.Errorf("read layout: %w", err)
		}
		if len(bytes.TrimSpace(data)) > 0 {
			return s.layoutJSONFromRaw(data)
		}
	}

	layout := buildScaffoldLayout(s.cfg)
	b, err := json.Marshal(layout)
	if err != nil {
		return "", fmt.Errorf("marshal scaffold: %w", err)
	}
	return string(b), nil
}

// layoutJSONFromRaw parses raw JSON/YAML, extracts a data context from a
// top-level "data" key, resolves template expressions, and returns the
// marshaled layout JSON.
func (s *Server) layoutJSONFromRaw(raw []byte) (string, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		// Not valid JSON — try YAML.
		if yamlErr := yaml.Unmarshal(raw, &obj); yamlErr != nil {
			return "", fmt.Errorf("parse layout: json: %v, yaml: %v", err, yamlErr)
		}
	}
	if len(obj) == 0 {
		return string(raw), nil
	}

	layoutJSON, _, err := layoutJSONFromObjectOrDefault(obj, s.cfg, nil)
	if err != nil {
		return "", err
	}
	return layoutJSON, nil
}

// renderWithChrome drives Chrome headless to render the layout.
func (s *Server) renderWithChrome(ctx context.Context, layoutJSON string) (*RenderResult, error) {
	return renderWithChrome(ctx, s.allocatorCtx, layoutJSON, defaultHTTPRenderOptions(s.cfg))
}

func defaultHTTPRenderOptions(cfg Config) RenderOptions {
	return defaultRenderOptions(fmt.Sprintf("http://localhost:%d", cfg.Port))
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
	if o.Threshold == 0 {
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

	renderCtx, renderCancel := context.WithTimeout(tabCtx, 30*time.Second)
	defer renderCancel()

	var screenshotBuf []byte
	var metrics RenderMetrics

	layoutArg, err := json.Marshal(layoutJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal layout argument: %w", err)
	}

	captureJS, err := injectCaptureCSSJS(captureCSS())
	if err != nil {
		return nil, err
	}

	actions := []chromedp.Action{
		chromedp.EmulateViewport(int64(opts.ViewportWidth), int64(opts.ViewportHeight)),
		chromedp.Navigate(opts.BaseURL + "/almanach"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Poll(`window.almanachReady === true`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(fmt.Sprintf(`window.almanachLoadLayout(JSON.parse(%s))`, layoutArg), nil),
		chromedp.Evaluate(waitForFontsAndFramesJS(), nil),
		chromedp.Sleep(opts.WaitAfterLoad),
		chromedp.Evaluate(captureJS, nil),
		chromedp.Evaluate(waitForFontsAndFramesJS(), nil),
	}
	if opts.CollectMetrics {
		actions = append(actions, chromedp.Evaluate(collectMetricsJS(), &metrics))
	}
	actions = append(actions,
		chromedp.Screenshot(opts.Selector, &screenshotBuf, chromedp.ByQuery, chromedp.NodeVisible),
	)

	if err := chromedp.Run(renderCtx, actions...); err != nil {
		log.Printf("[render] Chrome error: %v", err)
		return nil, fmt.Errorf("chrome render: %w", err)
	}

	log.Printf("[render] Screenshot captured: %d bytes PNG", len(screenshotBuf))

	bitmap, err := PngToBitmap(screenshotBuf, opts.Threshold)
	if err != nil {
		return nil, fmt.Errorf("bitmap convert: %w", err)
	}

	result := &RenderResult{
		Bitmap:     bitmap,
		PNG:        screenshotBuf,
		Theme:      extractThemeFromLayout(layoutJSON),
		RenderedAt: time.Now().UTC().Format(time.RFC3339),
		LayoutJSON: layoutJSON,
		Metrics:    metrics,
		Selector:   opts.Selector,
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
	return newChromeAllocatorWithViewport(cfg, defaultRenderViewportWidth, defaultRenderViewportHeight)
}

func newChromeAllocatorWithViewport(cfg Config, viewportWidth, viewportHeight int) (context.Context, context.CancelFunc) {
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

	log.Printf("Chrome mode: local (launching Chrome)")
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("force-device-scale-factor", "1.0"),
		chromedp.WindowSize(viewportWidth, viewportHeight),
	}

	// Disable font anti-aliasing in the render browser so text rasterizes
	// monochrome and small strokes survive the 1-bit conversion (ALMANACH-PIXELFONT).
	if env := renderFontEnv(); len(env) > 0 {
		opts = append(opts, chromedp.Env(env...))
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
