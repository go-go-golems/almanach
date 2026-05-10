package app

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLayoutJSONFromPathYAML(t *testing.T) {
	dir := t.TempDir()
	layoutPath := filepath.Join(dir, "layout.yaml")
	if err := os.WriteFile(layoutPath, []byte(`
almanach_studio_version: 1
theme: minimal
paperWidth: 384
bodyScale: 1.2
feedLines: 3
blocks:
  - id: title
    type: title
    data:
      text: Test
      subtitle: Standalone YAML
`), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := layoutJSONFromPathOrDefault(layoutPath, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if res.SourceKind != "file" {
		t.Fatalf("expected file source, got %q", res.SourceKind)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(res.LayoutJSON), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["theme"] != "minimal" {
		t.Fatalf("unexpected theme: %#v", obj["theme"])
	}
}

func TestZipBundleInlinesRelativeImage(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "layout.zip")
	writeTestZip(t, zipPath, map[string]string{
		"layout.yaml": `
almanach_studio_version: 1
theme: minimal
paperWidth: 384
bodyScale: 1.2
feedLines: 3
blocks:
  - id: img
    type: image
    data:
      src: images/fox.png
      height: 40
`,
		"images/fox.png": "\x89PNG\r\n\x1a\nnot-really-a-png-but-detectable-by-extension",
	})

	res, err := layoutJSONFromPathOrDefault(zipPath, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if res.SourceKind != "zip" || res.LayoutMember != "layout.yaml" {
		t.Fatalf("unexpected source metadata: %#v", res)
	}
	if !strings.Contains(res.LayoutJSON, "data:image/png;base64,") {
		t.Fatalf("expected inlined png data URL, got %s", res.LayoutJSON)
	}
	if strings.Contains(res.LayoutJSON, "images/fox.png") {
		t.Fatalf("relative image path was not replaced: %s", res.LayoutJSON)
	}
}

func TestZipBundlePreservesDataAndRemoteImages(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "layout.zip")
	writeTestZip(t, zipPath, map[string]string{
		"layout.yaml": `
almanach_studio_version: 1
theme: minimal
blocks:
  - id: data
    type: image
    data:
      src: data:image/png;base64,AAAA
  - id: remote
    type: image
    data:
      src: https://example.com/foo.png
`,
	})

	res, err := layoutJSONFromPathOrDefault(zipPath, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.LayoutJSON, "data:image/png;base64,AAAA") {
		t.Fatalf("data URL not preserved: %s", res.LayoutJSON)
	}
	if !strings.Contains(res.LayoutJSON, "https://example.com/foo.png") {
		t.Fatalf("remote URL not preserved: %s", res.LayoutJSON)
	}
}

func TestZipBundleAmbiguousLayoutCandidates(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "layout.zip")
	writeTestZip(t, zipPath, map[string]string{
		"one.yaml": "blocks: []\n",
		"two.yaml": "blocks: []\n",
	})

	_, err := layoutJSONFromPathOrDefault(zipPath, Config{})
	if err == nil || !strings.Contains(err.Error(), "multiple root-level layout candidates") {
		t.Fatalf("expected ambiguous layout error, got %v", err)
	}
}

func writeTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()
	out, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}
