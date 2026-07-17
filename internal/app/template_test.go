package app

import (
	"strings"
	"testing"
)

func TestResolveValue_SimpleSubstitution(t *testing.T) {
	ctx := map[string]string{"name": "world"}
	got, err := resolveValue("hello {{name}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("got %q, want %q", got, "hello world")
	}
}

func TestResolveValue_MissingKeyWithoutFallback(t *testing.T) {
	ctx := map[string]string{}
	_, err := resolveValue("hello {{missing}}", ctx)
	if err == nil {
		t.Fatal("expected error for missing key without fallback")
	}
}

func TestResolveValue_MissingKeyWithFallback(t *testing.T) {
	ctx := map[string]string{}
	got, err := resolveValue("hello {{missing:friend}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello friend" {
		t.Fatalf("got %q, want %q", got, "hello friend")
	}
}

// Environment variables must never resolve from a layout, even when set in the
// process environment. A layout is passive data; {{$SECRET}} reaching prints or
// debug artifacts would be an exfiltration channel (ALMANACH-WORKSLIP Phase 1).
func TestResolveValue_EnvVarSyntaxDoesNotResolve(t *testing.T) {
	t.Setenv("ALMANACH_TEST_VAR", "from-env")

	ctx := map[string]string{}
	_, err := resolveValue("value: {{$ALMANACH_TEST_VAR}}", ctx)
	if err == nil {
		t.Fatal("expected error: $ENV expressions must not resolve from the environment")
	}
	if strings.Contains(err.Error(), "from-env") {
		t.Fatalf("error must not leak the env value: %v", err)
	}
}

// A $-prefixed expression with a fallback behaves like any unknown key: the
// fallback wins, the environment is never consulted.
func TestResolveValue_EnvVarSyntaxUsesFallbackNotEnv(t *testing.T) {
	t.Setenv("ALMANACH_TEST_NONEXISTENT", "must-not-appear")
	ctx := map[string]string{}
	got, err := resolveValue("hello {{$ALMANACH_TEST_NONEXISTENT:anonymous}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello anonymous" {
		t.Fatalf("got %q, want %q", got, "hello anonymous")
	}
}

func TestResolveValue_MultipleExpressions(t *testing.T) {
	ctx := map[string]string{"a": "alpha", "b": "beta"}
	got, err := resolveValue("{{a}} and {{b}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "alpha and beta" {
		t.Fatalf("got %q, want %q", got, "alpha and beta")
	}
}

func TestResolveValue_FallbackWithColons(t *testing.T) {
	ctx := map[string]string{}
	got, err := resolveValue("{{key:a:b:c}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a:b:c" {
		t.Fatalf("got %q, want %q", got, "a:b:c")
	}
}

func TestResolveValue_UnclosedExpression(t *testing.T) {
	ctx := map[string]string{}
	_, err := resolveValue("{{no closing", ctx)
	if err == nil {
		t.Fatal("expected error for unclosed {{")
	}
}

func TestResolveValue_EmptyExpression(t *testing.T) {
	ctx := map[string]string{}
	_, err := resolveValue("{{}}", ctx)
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
}

func TestResolveValue_NoExpressions(t *testing.T) {
	ctx := map[string]string{"name": "world"}
	got, err := resolveValue("plain text", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "plain text" {
		t.Fatalf("got %q, want %q", got, "plain text")
	}
}

// A data-context key that happens to start with $ still resolves — only the
// implicit environment lookup is gone, not the character.
func TestResolveValue_DollarKeyResolvesFromContext(t *testing.T) {
	t.Setenv("HOME_SWEET", "env-value")
	ctx := map[string]string{"$HOME_SWEET": "ctx-value"}
	got, err := resolveValue("{{$HOME_SWEET}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ctx-value" {
		t.Fatalf("got %q, want %q", got, "ctx-value")
	}
}

func TestWalkValue_NestedMap(t *testing.T) {
	obj := map[string]interface{}{
		"data": map[string]interface{}{
			"text": "{{title}}",
		},
	}
	ctx := map[string]string{"title": "HELLO"}
	err := walkMap(obj, ctx)
	if err != nil {
		t.Fatal(err)
	}
	data := obj["data"].(map[string]interface{})
	if data["text"] != "HELLO" {
		t.Fatalf("got %q, want %q", data["text"], "HELLO")
	}
}

func TestWalkValue_SliceOfMaps(t *testing.T) {
	obj := map[string]interface{}{
		"blocks": []interface{}{
			map[string]interface{}{
				"type": "title",
				"data": map[string]interface{}{
					"text": "{{title}}",
				},
			},
		},
	}
	ctx := map[string]string{"title": "MORNING"}
	err := walkMap(obj, ctx)
	if err != nil {
		t.Fatal(err)
	}
	blocks := obj["blocks"].([]interface{})
	block := blocks[0].(map[string]interface{})
	data := block["data"].(map[string]interface{})
	if data["text"] != "MORNING" {
		t.Fatalf("got %q, want %q", data["text"], "MORNING")
	}
}

func TestResolveTemplate_NoOpOnEmptyContext(t *testing.T) {
	obj := map[string]interface{}{
		"text": "{{should_not_resolve}}",
	}
	err := ResolveTemplate(obj, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	// String should remain unchanged
	if obj["text"] != "{{should_not_resolve}}" {
		t.Fatalf("expected no-op, got %q", obj["text"])
	}
}

func TestResolveTemplate_FullLayout(t *testing.T) {
	obj := map[string]interface{}{
		"theme": "minimal",
		"blocks": []interface{}{
			map[string]interface{}{
				"id":   "title-1",
				"type": "title",
				"data": map[string]interface{}{
					"text":     "{{title}}",
					"subtitle": "{{subtitle:default subtitle}}",
				},
			},
			map[string]interface{}{
				"id":   "date-1",
				"type": "date",
				"data": map[string]interface{}{
					"date": "{{date}}",
					"day":  "{{day}}",
				},
			},
		},
	}
	ctx := map[string]string{
		"title": "MORNING SIGNAL",
		"date":  "May 26, 2026",
		"day":   "Monday",
	}
	err := ResolveTemplate(obj, ctx)
	if err != nil {
		t.Fatal(err)
	}

	blocks := obj["blocks"].([]interface{})
	titleData := blocks[0].(map[string]interface{})["data"].(map[string]interface{})
	if titleData["text"] != "MORNING SIGNAL" {
		t.Fatalf("got %q", titleData["text"])
	}
	if titleData["subtitle"] != "default subtitle" {
		t.Fatalf("got %q", titleData["subtitle"])
	}
	dateData := blocks[1].(map[string]interface{})["data"].(map[string]interface{})
	if dateData["date"] != "May 26, 2026" {
		t.Fatalf("got %q", dateData["date"])
	}
}

func TestResolveTemplate_MissingVariableError(t *testing.T) {
	obj := map[string]interface{}{
		"text": "{{missing_var}}",
	}
	ctx := map[string]string{"other": "value"}
	err := ResolveTemplate(obj, ctx)
	if err == nil {
		t.Fatal("expected error for missing variable")
	}
	// Should mention the variable name and suggest --data or -D
	if !strings.Contains(err.Error(), "missing_var") {
		t.Fatalf("error should mention missing_var: %v", err)
	}
	if !strings.Contains(err.Error(), "--data") {
		t.Fatalf("error should suggest --data: %v", err)
	}
}

func TestResolveTemplate_NonStringValuesUnchanged(t *testing.T) {
	obj := map[string]interface{}{
		"paperWidth": 384,
		"bodyScale":  1.45,
		"feedLines":  3,
		"active":     true,
		"empty":      nil,
	}
	ctx := map[string]string{"anything": "value"}
	err := ResolveTemplate(obj, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if obj["paperWidth"] != 384 {
		t.Fatalf("numeric value changed: %v", obj["paperWidth"])
	}
	if obj["bodyScale"] != 1.45 {
		t.Fatalf("float value changed: %v", obj["bodyScale"])
	}
	if obj["feedLines"] != 3 {
		t.Fatalf("int value changed: %v", obj["feedLines"])
	}
	if obj["active"] != true {
		t.Fatalf("bool value changed: %v", obj["active"])
	}
	if obj["empty"] != nil {
		t.Fatalf("nil value changed: %v", obj["empty"])
	}
}
