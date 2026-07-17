package layoutpb

import (
	"os"
	"path/filepath"
	"testing"

	layoutv1 "github.com/go-go-golems/almanach/gen/almanach/layout/v1"
	"google.golang.org/protobuf/proto"
)

// goldenPath is the shared wire fixture the TS decode test also reads, so both
// sides lock the same contract.
var goldenPath = filepath.Join("..", "..", "proto", "almanach", "layout", "v1", "testdata", "layout_golden.json")

func loadGolden(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	return data
}

// TestGoldenDecode locks the field-level meaning of the wire format: the golden
// JSON must decode into the values we expect (enums, Struct content, nested
// messages, maps, optional scalars).
func TestGoldenDecode(t *testing.T) {
	layout, err := UnmarshalJSON(loadGolden(t))
	if err != nil {
		t.Fatalf("decode golden: %v", err)
	}

	if got := layout.GetSchemaVersion(); got != SchemaVersionV1 {
		t.Errorf("schemaVersion = %d, want %d", got, SchemaVersionV1)
	}
	if got := layout.GetPaperWidth(); got != 384 {
		t.Errorf("paperWidth = %d, want 384", got)
	}
	if got := layout.GetTheme(); got != "minimal" {
		t.Errorf("theme = %q, want minimal", got)
	}
	if got := layout.GetBodyScale(); got != 1.35 {
		t.Errorf("bodyScale = %v, want 1.35", got)
	}
	if got := layout.GetMargin().GetLeft(); got != 6 {
		t.Errorf("margin.left = %d, want 6", got)
	}
	if got := layout.GetMargin().GetTop(); got != 10 {
		t.Errorf("margin.top = %d, want 10", got)
	}
	if got := len(layout.GetBlocks()); got != 3 {
		t.Fatalf("blocks len = %d, want 3", got)
	}

	// Typography preset with an enum-valued text_case and optional scalars.
	title := layout.GetTypography().GetPresets()["title"]
	if title == nil {
		t.Fatal("missing 'title' preset")
	}
	if got := title.GetTextCase(); got != layoutv1.TextCase_TEXT_CASE_UPPER {
		t.Errorf("title.textCase = %v, want TEXT_CASE_UPPER", got)
	}
	if got := title.GetWeight(); got != 700 {
		t.Errorf("title.weight = %d, want 700", got)
	}

	// Block with per-block render override + Struct content.
	quote := layout.GetBlocks()[1]
	if quote.GetType() != "quote" {
		t.Errorf("blocks[1].type = %q, want quote", quote.GetType())
	}
	if got := quote.GetRender().GetPrinterDensity(); got != 38 {
		t.Errorf("quote.render.printerDensity = %d, want 38", got)
	}
	if got := quote.GetRender().GetRasterMode(); got != layoutv1.RasterMode_RASTER_MODE_THRESHOLD {
		t.Errorf("quote.render.rasterMode = %v, want THRESHOLD", got)
	}
	if got := quote.GetContent().GetFields()["author"].GetStringValue(); got != "Dorothy Parker" {
		t.Errorf("quote.content.author = %q, want Dorothy Parker", got)
	}

	img := layout.GetBlocks()[2]
	if got := img.GetRender().GetRasterMode(); got != layoutv1.RasterMode_RASTER_MODE_ATKINSON {
		t.Errorf("img.render.rasterMode = %v, want ATKINSON", got)
	}
	if got := img.GetRender().GetGamma(); got != 0.8 {
		t.Errorf("img.render.gamma = %v, want 0.8", got)
	}
}

// TestRoundTrip proves encode/decode is stable: decoding, re-encoding, and
// decoding again yields an equal message. This is the contract-stability guard.
func TestRoundTrip(t *testing.T) {
	first, err := UnmarshalJSON(loadGolden(t))
	if err != nil {
		t.Fatalf("first decode: %v", err)
	}
	encoded, err := MarshalJSON(first)
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	second, err := UnmarshalJSON(encoded)
	if err != nil {
		t.Fatalf("second decode: %v", err)
	}
	if !proto.Equal(first, second) {
		t.Errorf("round-trip mismatch:\n first = %v\nsecond = %v", first, second)
	}
}

// TestBinaryRoundTrip covers the compact wire form too.
func TestBinaryRoundTrip(t *testing.T) {
	layout, err := UnmarshalJSON(loadGolden(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	bin, err := MarshalBinary(layout)
	if err != nil {
		t.Fatalf("marshal binary: %v", err)
	}
	back, err := UnmarshalBinary(bin)
	if err != nil {
		t.Fatalf("unmarshal binary: %v", err)
	}
	if !proto.Equal(layout, back) {
		t.Error("binary round-trip mismatch")
	}
}
