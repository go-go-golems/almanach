package app

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// wikipediaOnThisDay represents the Wikipedia API response.
type wikipediaOnThisDay []struct {
	Text  string `json:"text"`
	Year  int    `json:"year"`
	Pages []struct {
		Title string `json:"title"`
	} `json:"pages"`
}

// fetchHistory fetches a historical event that happened on today's date.
// Uses the Wikimedia "On this day" API.
// Returns nil if the API is unavailable.
func fetchHistory(t time.Time) *HistoryData {
	url := fmt.Sprintf("https://api.wikimedia.org/feed/v1/wikipedia/en/onthisday/all/%02d/%02d", t.Month(), t.Day())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fallbackHistory(t)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fallbackHistory(t)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallbackHistory(t)
	}

	// The response is a map of event types to arrays
	var events map[string]wikipediaOnThisDay
	if err := json.Unmarshal(body, &events); err != nil {
		return fallbackHistory(t)
	}

	// "events" type has the most relevant entries
	evts, ok := events["events"]
	if !ok || len(evts) == 0 {
		// Try "selected" as fallback
		evts, ok = events["selected"]
		if !ok || len(evts) == 0 {
			return fallbackHistory(t)
		}
	}

	pick := evts[rand.Intn(len(evts))]
	return &HistoryData{
		Label: "Today in History",
		Items: []HistoryItem{{
			Year:  fmt.Sprintf("%d", pick.Year),
			Event: pick.Text,
		}},
	}
}

var fallbackHistories = []HistoryItem{
	{Year: "1969", Event: "Apollo 11 lands on the Moon. Neil Armstrong becomes the first human to walk on the lunar surface."},
	{Year: "1453", Event: "Constantinople falls to the Ottoman Empire, marking the end of the Byzantine Empire."},
	{Year: "1989", Event: "The Berlin Wall falls, marking the beginning of the end of the Cold War."},
}

func fallbackHistory(t time.Time) *HistoryData {
	return &HistoryData{
		Label: "Today in History",
		Items: []HistoryItem{fallbackHistories[rand.Intn(len(fallbackHistories))]},
	}
}
