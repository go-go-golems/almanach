package app

// newsFeed is a hardcoded fallback for development.
// Phase 4d will replace this with RSS parsing.

var fallbackHeadlines = []NewsItem{
	{Headline: "Placeholder headline 1", Source: "BBC", Time: "now"},
	{Headline: "Placeholder headline 2", Source: "Reuters", Time: "now"},
}

// fetchNews returns placeholder news items.
// TODO: Implement RSS parsing for BBC, Reuters, etc.
func fetchNews() *NewsData {
	return &NewsData{Label: "Top News", Items: fallbackHeadlines}
}
