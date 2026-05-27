package app

import (
	"encoding/json"
	"fmt"
	"time"
)

// ImageData holds image block fields. Kept here alongside the other data types
// so the Go type definitions stay in one place.
type ImageData struct {
	Label     string `json:"label"`
	Src       string `json:"src"`
	Alt       string `json:"alt,omitempty"`
	Caption   string `json:"caption,omitempty"`
	Height    int    `json:"height,omitempty"`
	Fit       string `json:"fit,omitempty"`
	Border    bool   `json:"border,omitempty"`
	Grayscale bool   `json:"grayscale,omitempty"`
}

// Layout represents the full almanac page layout sent to the SPA.
//
// Keep this schema aligned with web/src/almanach-studio.jsx:
// DEFAULTS, BLOCK_TYPES, RENDERERS, buildLayoutJson(), and parseLayoutJson().
type Layout struct {
	Version    int     `json:"almanach_studio_version"`
	ExportedAt string  `json:"exported_at"`
	Theme      string  `json:"theme"`
	PaperWidth int     `json:"paperWidth"`
	BodyScale  float64 `json:"bodyScale"`
	FeedLines  int     `json:"feedLines"`
	Blocks     []Block `json:"blocks"`
}

// Block represents a single almanac block (title, weather, news, etc.).
type Block struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Data types for each block type. These intentionally mirror the React
// renderer's block data shape, not an independent Go-domain schema.

type TitleData struct {
	Text     string `json:"text"`
	Subtitle string `json:"subtitle"`
}

type DateData struct {
	Date string `json:"date"`
	Day  string `json:"day"`
}

type WeatherData struct {
	Temp      string `json:"temp"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Condition string `json:"condition"`
	Sunrise   string `json:"sunrise,omitempty"`
	Sunset    string `json:"sunset,omitempty"`
	Humidity  string `json:"humidity,omitempty"`
	Wind      string `json:"wind,omitempty"`
}

type NewsData struct {
	Label string     `json:"label"`
	Items []NewsItem `json:"items"`
}

type NewsItem struct {
	Headline string `json:"headline"`
	Source   string `json:"source"`
	Time     string `json:"time,omitempty"`
}

type PlanData struct {
	Label string     `json:"label"`
	Items []PlanItem `json:"items"`
}

type PlanItem struct {
	Time string `json:"time"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type QuoteData struct {
	Label  string `json:"label"`
	Text   string `json:"text"`
	Author string `json:"author"`
	Source string `json:"source,omitempty"`
}

type WordData struct {
	Label      string `json:"label"`
	Word       string `json:"word"`
	Phonetic   string `json:"phonetic,omitempty"`
	Part       string `json:"part"`
	Definition string `json:"definition"`
	Example    string `json:"example,omitempty"`
}

type HistoryData struct {
	Label string        `json:"label"`
	Items []HistoryItem `json:"items"`
}

type HistoryItem struct {
	Year  string `json:"year"`
	Event string `json:"event"`
}

type HabitsData struct {
	Label      string      `json:"label"`
	Range      string      `json:"range,omitempty"`
	Columns    []string    `json:"columns,omitempty"`
	Items      []HabitItem `json:"items"`
	Reflection string      `json:"reflection,omitempty"`
}

type HabitItem struct {
	Name string `json:"name"`
	Done bool   `json:"done,omitempty"`
	Days []int  `json:"days,omitempty"`
}

type DidData struct {
	Label string   `json:"label"`
	Items []string `json:"items"`
}

type NoteData struct {
	Label  string `json:"label"`
	Text   string `json:"text"`
	Author string `json:"author,omitempty"`
}

type MoodData struct {
	Label  string `json:"label"`
	Mood   int    `json:"mood"`
	Energy int    `json:"energy"`
	Sleep  string `json:"sleep,omitempty"`
	Notes  string `json:"notes,omitempty"`
}

type ReadingData struct {
	Label   string            `json:"label"`
	Current ReadingCurrent    `json:"current"`
	Next    []string          `json:"next,omitempty"`
	Extra   map[string]string `json:"extra,omitempty"`
}

type ReadingCurrent struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Progress int    `json:"progress"`
}

type ReflectionData struct {
	Label   string `json:"label"`
	Well    string `json:"well"`
	Better  string `json:"better"`
	Learned string `json:"learned"`
	Quote   string `json:"quote,omitempty"`
}

// blockID counter for generating unique block IDs.
var blockCounter int

func nextBlockID() string {
	blockCounter++
	return fmt.Sprintf("b%d", blockCounter)
}

// newBlock creates a Block with a generated ID and JSON-encoded data.
func newBlock(typ string, data any) Block {
	raw, _ := json.Marshal(data)
	return Block{ID: nextBlockID(), Type: typ, Data: raw}
}

// buildScaffoldLayout produces a minimal layout with just a title and date block.
// Used when no layout file is provided — no content is invented, no APIs are called.
func buildScaffoldLayout(cfg Config) *Layout {
	now := time.Now()
	return &Layout{
		Version:    1,
		ExportedAt: now.UTC().Format(time.RFC3339),
		Theme:      cfg.DefaultTheme,
		PaperWidth: cfg.PaperWidth,
		BodyScale:  cfg.BodyScale,
		FeedLines:  cfg.FeedLines,
		Blocks: []Block{
			newBlock("title", TitleData{
				Text:     "ALMANACH",
				Subtitle: now.Format("January 2, 2006"),
			}),
			newBlock("date", DateData{
				Date: now.Format("January 2, 2006"),
				Day:  now.Format("Monday"),
			}),
		},
	}
}
