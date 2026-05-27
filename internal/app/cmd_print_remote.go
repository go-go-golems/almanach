package app

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

const defaultRemotePrintURL = "https://almanach.crib.scapegoat.dev/api/render-and-print"
const defaultRemoteRenderURL = "https://almanach.crib.scapegoat.dev/api/render"

type PrintRemoteCommand struct {
	*cmds.CommandDescription
}

type PrintRemoteSettings struct {
	Layout             string `glazed:"layout"`
	URL                string `glazed:"url"`
	DryRun             bool   `glazed:"dry-run"`
	TimeoutSeconds     int    `glazed:"timeout-seconds"`
	InsecureSkipVerify bool   `glazed:"insecure-skip-verify"`
	Data               string `glazed:"data"`
	Define             string `glazed:"define"`
}

func newPrintRemoteCommand() (*PrintRemoteCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"print-remote",
		cmds.WithShort("Send an Almanach layout to the remote crib render-and-print service"),
		cmds.WithLong(`Load a JSON/YAML layout or ZIP layout bundle locally, convert it to the layout JSON expected by the Almanach HTTP API, and POST it to the remote render service.

This command is a CLI wrapper for https://almanach.crib.scapegoat.dev/api/render-and-print. It avoids hand-written YAML-to-JSON shell pipelines and also supports ZIP layout bundles with local image assets.

Examples:
  almanach-render-service print-remote --layout daily.yaml
  almanach-render-service print-remote --layout layout-bundle.zip --output yaml
  almanach-render-service print-remote --layout daily.yaml --dry-run --output yaml
  almanach-render-service print-remote --layout daily.yaml --url https://almanach.crib.scapegoat.dev/api/render-and-print
`),
		cmds.WithFlags(
			fields.New("layout", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Layout file or ZIP bundle to send. Accepts JSON, YAML, or .zip. Empty uses the generated default layout.")),
			fields.New("url", fields.TypeString, fields.WithDefault(defaultRemotePrintURL), fields.WithHelp("Remote Almanach API URL. Defaults to the crib render-and-print endpoint.")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("POST to the render endpoint instead of render-and-print when using the default URL")),
			fields.New("timeout-seconds", fields.TypeInteger, fields.WithDefault(90), fields.WithHelp("HTTP request timeout in seconds")),
			fields.New("insecure-skip-verify", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Skip TLS certificate verification for self-signed development endpoints")),
			fields.New("data", fields.TypeString, fields.WithDefault(""), fields.WithHelp("YAML/JSON data context file for template resolution")),
			fields.New("define", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Inline key=value for template variables (comma-separated: -D key=val,key2=val2)")),
		),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)
	return &PrintRemoteCommand{CommandDescription: desc}, nil
}

func (c *PrintRemoteCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	s := &PrintRemoteSettings{}
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

	url := strings.TrimSpace(s.URL)
	if url == "" {
		url = defaultRemotePrintURL
	}
	if s.DryRun && url == defaultRemotePrintURL {
		url = defaultRemoteRenderURL
	}

	response, err := postLayoutToRemoteAlmanach(ctx, remotePostRequest{
		URL:                url,
		LayoutJSON:         layoutSource.LayoutJSON,
		Timeout:            time.Duration(s.TimeoutSeconds) * time.Second,
		InsecureSkipVerify: s.InsecureSkipVerify,
	})
	if err != nil {
		return err
	}

	return gp.AddRow(ctx, types.NewRow(
		types.MRP("ok", response.OK),
		types.MRP("printed", response.Printed),
		types.MRP("dry_run", s.DryRun),
		types.MRP("url", url),
		types.MRP("status_code", response.StatusCode),
		types.MRP("width", response.Width),
		types.MRP("height", response.Height),
		types.MRP("rendered_at", response.RenderedAt),
		types.MRP("source_kind", layoutSource.SourceKind),
		types.MRP("layout_member", layoutSource.LayoutMember),
		types.MRP("printer_response", response.PrinterResponse),
		types.MRP("response", response.Body),
	))
}

type remotePostRequest struct {
	URL                string
	LayoutJSON         string
	Timeout            time.Duration
	InsecureSkipVerify bool
}

type remotePostResponse struct {
	OK              bool
	Printed         bool
	StatusCode      int
	Width           any
	Height          any
	RenderedAt      any
	PrinterResponse any
	Body            map[string]any
}

func postLayoutToRemoteAlmanach(ctx context.Context, req remotePostRequest) (*remotePostResponse, error) {
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("remote almanach url is required")
	}
	if strings.TrimSpace(req.LayoutJSON) == "" {
		return nil, fmt.Errorf("layout JSON is empty")
	}
	if req.Timeout <= 0 {
		req.Timeout = 90 * time.Second
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.URL, bytes.NewBufferString(req.LayoutJSON))
	if err != nil {
		return nil, fmt.Errorf("create remote almanach request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	transport := http.DefaultTransport
	if req.InsecureSkipVerify {
		transport = &http.Transport{
			// #nosec G402 -- Explicit CLI flag for self-signed development endpoints.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	client := &http.Client{Timeout: req.Timeout, Transport: transport}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("post remote almanach layout: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read remote almanach response: %w", err)
	}

	var body map[string]any
	if len(bytes.TrimSpace(bodyBytes)) > 0 {
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			return nil, fmt.Errorf("decode remote almanach response %d: %w: %s", resp.StatusCode, err, string(bodyBytes))
		}
	}
	if body == nil {
		body = map[string]any{}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if msg, ok := body["error"].(string); ok && msg != "" {
			return nil, fmt.Errorf("remote almanach returned %d: %s", resp.StatusCode, msg)
		}
		return nil, fmt.Errorf("remote almanach returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return &remotePostResponse{
		OK:              boolFromMap(body, "ok"),
		Printed:         boolFromMap(body, "printed"),
		StatusCode:      resp.StatusCode,
		Width:           body["width"],
		Height:          body["height"],
		RenderedAt:      body["renderedAt"],
		PrinterResponse: body["printerResponse"],
		Body:            body,
	}, nil
}

func boolFromMap(m map[string]any, key string) bool {
	v, ok := m[key].(bool)
	return ok && v
}
