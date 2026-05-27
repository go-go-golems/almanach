package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDataContext_Empty(t *testing.T) {
	ctx, err := LoadDataContext("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx) != 0 {
		t.Fatalf("expected empty context, got %d entries", len(ctx))
	}
}

func TestLoadDataContext_YAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.yaml")
	err := os.WriteFile(path, []byte("title: HELLO\nday: Monday\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := LoadDataContext(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ctx["title"] != "HELLO" {
		t.Fatalf("got %q", ctx["title"])
	}
	if ctx["day"] != "Monday" {
		t.Fatalf("got %q", ctx["day"])
	}
}

func TestLoadDataContext_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	err := os.WriteFile(path, []byte(`{"title":"HELLO","day":"Monday"}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := LoadDataContext(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ctx["title"] != "HELLO" {
		t.Fatalf("got %q", ctx["title"])
	}
	if ctx["day"] != "Monday" {
		t.Fatalf("got %q", ctx["day"])
	}
}

func TestLoadDataContext_DefinesOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.yaml")
	err := os.WriteFile(path, []byte("title: FROM_FILE\nsubtitle: kept\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := LoadDataContext(path, []string{"title=FROM_DEFINE", "extra=added"})
	if err != nil {
		t.Fatal(err)
	}

	if ctx["title"] != "FROM_DEFINE" {
		t.Fatalf("define should override file, got %q", ctx["title"])
	}
	if ctx["subtitle"] != "kept" {
		t.Fatalf("file value should be kept, got %q", ctx["subtitle"])
	}
	if ctx["extra"] != "added" {
		t.Fatalf("define should add new key, got %q", ctx["extra"])
	}
}

func TestLoadDataContext_DefinesOnly(t *testing.T) {
	ctx, err := LoadDataContext("", []string{"key1=val1", "key2=val2"})
	if err != nil {
		t.Fatal(err)
	}
	if ctx["key1"] != "val1" {
		t.Fatalf("got %q", ctx["key1"])
	}
	if ctx["key2"] != "val2" {
		t.Fatalf("got %q", ctx["key2"])
	}
}

func TestLoadDataContext_InvalidDefine(t *testing.T) {
	_, err := LoadDataContext("", []string{"no_equals_sign"})
	if err == nil {
		t.Fatal("expected error for invalid define")
	}
}

func TestLoadDataContext_EmptyDefineKey(t *testing.T) {
	_, err := LoadDataContext("", []string{"=value"})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestLoadDataContext_MissingFile(t *testing.T) {
	_, err := LoadDataContext("/nonexistent/data.yaml", nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadDataContext_NumericValuesFlattened(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.yaml")
	err := os.WriteFile(path, []byte("count: 42\npi: 3.14\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := LoadDataContext(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ctx["count"] != "42" {
		t.Fatalf("got %q", ctx["count"])
	}
	if ctx["pi"] != "3.14" {
		t.Fatalf("got %q", ctx["pi"])
	}
}
