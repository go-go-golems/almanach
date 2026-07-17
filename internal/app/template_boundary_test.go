package app

import (
	"strings"
	"testing"
)

// A layout can self-activate template resolution by carrying its own `data:`
// map (layoutJSONFromObjectOrDefault merges it into the context). These tests
// pin the security boundary: even then, the process environment must never be
// readable from a layout. Before ALMANACH-WORKSLIP Phase 1, {{$NAME}} resolved
// via os.LookupEnv, which let any layout — including one POSTed to the HTTP
// server — exfiltrate env vars into rendered output.
func TestLayoutObject_SelfActivatedDataCannotReadEnv(t *testing.T) {
	t.Setenv("ALMANACH_BOUNDARY_SECRET", "leaked")

	obj := map[string]interface{}{
		"theme": "minimal",
		// Non-empty data map activates ResolveTemplate on this layout.
		"data": map[string]interface{}{"greeting": "hello"},
		"blocks": []interface{}{
			map[string]interface{}{
				"id":   "t",
				"type": "title",
				"data": map[string]interface{}{
					"text": "{{$ALMANACH_BOUNDARY_SECRET}}",
				},
			},
		},
	}

	_, _, err := layoutJSONFromObjectOrDefault(obj, LoadConfig(), nil)
	if err == nil {
		t.Fatal("expected error: $-expressions must not resolve from the environment")
	}
	if strings.Contains(err.Error(), "leaked") {
		t.Fatalf("error must not contain the env value: %v", err)
	}
}

// Without any data context (no --data/--define, no layout data map), template
// markers pass through untouched — the server path relies on this.
func TestLayoutObject_NoContextLeavesMarkersUnresolved(t *testing.T) {
	t.Setenv("ALMANACH_BOUNDARY_SECRET", "leaked")

	obj := map[string]interface{}{
		"theme": "minimal",
		"blocks": []interface{}{
			map[string]interface{}{
				"id":   "t",
				"type": "title",
				"data": map[string]interface{}{
					"text": "{{$ALMANACH_BOUNDARY_SECRET}} and {{unset_key}}",
				},
			},
		},
	}

	layoutJSON, _, err := layoutJSONFromObjectOrDefault(obj, LoadConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(layoutJSON, "leaked") {
		t.Fatalf("env value leaked into layout JSON: %s", layoutJSON)
	}
	if !strings.Contains(layoutJSON, "{{$ALMANACH_BOUNDARY_SECRET}}") ||
		!strings.Contains(layoutJSON, "{{unset_key}}") {
		t.Fatalf("markers should pass through unresolved, got: %s", layoutJSON)
	}
}
