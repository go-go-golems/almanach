package app

import (
	"encoding/json"
	"testing"
)

func TestFrontendBlockSchemaDataKeys(t *testing.T) {
	tests := []struct {
		name string
		b    Block
		want map[string]any
	}{
		{
			name: "title uses text key",
			b: newBlock("title", TitleData{
				Text:     "Daily Almanac",
				Subtitle: "Today",
			}),
			want: map[string]any{"text": "Daily Almanac", "subtitle": "Today"},
		},
		{
			name: "word uses part key",
			b: newBlock("word", WordData{
				Label:      "Word of the Day",
				Word:       "serendipity",
				Part:       "noun",
				Definition: "Happy accident.",
			}),
			want: map[string]any{"label": "Word of the Day", "word": "serendipity", "part": "noun", "definition": "Happy accident."},
		},
		{
			name: "history uses items list",
			b: newBlock("history", HistoryData{
				Label: "Today in History",
				Items: []HistoryItem{{Year: "1969", Event: "Moon landing"}},
			}),
			want: map[string]any{"label": "Today in History", "items": []any{map[string]any{"year": "1969", "event": "Moon landing"}}},
		},
		{
			name: "did uses items list",
			b: newBlock("did", DidData{
				Label: "Did You Know?",
				Items: []string{"Honey never spoils."},
			}),
			want: map[string]any{"label": "Did You Know?", "items": []any{"Honey never spoils."}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]any
			if err := json.Unmarshal(tt.b.Data, &got); err != nil {
				t.Fatalf("unmarshal block data: %v", err)
			}
			for k, want := range tt.want {
				gotValue, ok := got[k]
				if !ok {
					t.Fatalf("missing key %q in %#v", k, got)
				}
				wantJSON, _ := json.Marshal(want)
				gotJSON, _ := json.Marshal(gotValue)
				if string(gotJSON) != string(wantJSON) {
					t.Fatalf("key %q: got %s, want %s", k, gotJSON, wantJSON)
				}
			}
		})
	}
}

// TestBuildScaffoldLayout verifies the scaffold has the expected shape.
func TestBuildScaffoldLayout(t *testing.T) {
	cfg := Config{
		DefaultTheme: "minimal",
		PaperWidth:   384,
		BodyScale:    1.4,
		FeedLines:    3,
	}
	layout := buildScaffoldLayout(cfg)
	if layout.Version != 1 {
		t.Fatalf("expected version 1, got %d", layout.Version)
	}
	if len(layout.Blocks) != 2 {
		t.Fatalf("expected 2 blocks (title + date), got %d", len(layout.Blocks))
	}
	if layout.Blocks[0].Type != "title" {
		t.Fatalf("expected first block type 'title', got %q", layout.Blocks[0].Type)
	}
	if layout.Blocks[1].Type != "date" {
		t.Fatalf("expected second block type 'date', got %q", layout.Blocks[1].Type)
	}
	if layout.Theme != "minimal" {
		t.Fatalf("expected theme 'minimal', got %q", layout.Theme)
	}
}
