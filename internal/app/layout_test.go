package app

import (
	"encoding/json"
	"testing"
	"time"
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

func TestDefaultLocalFetchersUseFrontendSchema(t *testing.T) {
	if news := fetchNews(); news == nil || news.Label == "" || len(news.Items) == 0 || news.Items[0].Time == "" {
		t.Fatalf("fetchNews() did not produce frontend-shaped news data: %#v", news)
	}

	if quote := fetchQuote(); quote == nil || quote.Label == "" || quote.Text == "" || quote.Author == "" {
		t.Fatalf("fetchQuote() did not produce frontend-shaped quote data: %#v", quote)
	}

	if word := fetchWord(); word == nil || word.Label == "" || word.Word == "" || word.Part == "" || word.Definition == "" {
		t.Fatalf("fetchWord() did not produce frontend-shaped word data: %#v", word)
	}

	if history := fallbackHistory(time.Now()); history == nil || history.Label == "" || len(history.Items) == 0 || history.Items[0].Year == "" || history.Items[0].Event == "" {
		t.Fatalf("fallbackHistory() did not produce frontend-shaped history data: %#v", history)
	}
}
