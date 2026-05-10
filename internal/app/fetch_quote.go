package app

import (
	"math/rand"
)

// quotePool is a local pool of quotes for offline operation.
// Can be replaced with an API (e.g., quotable.io) in the future.
var quotePool = []QuoteData{
	{Label: "Quote of the Day", Text: "The unexamined life is not worth living.", Author: "Socrates", Source: "Apology"},
	{Label: "Quote of the Day", Text: "In the middle of difficulty lies opportunity.", Author: "Albert Einstein"},
	{Label: "Quote of the Day", Text: "The only way to do great work is to love what you do.", Author: "Steve Jobs"},
	{Label: "Quote of the Day", Text: "Simplicity is the ultimate sophistication.", Author: "Leonardo da Vinci"},
	{Label: "Quote of the Day", Text: "What we know is a drop, what we don't know is an ocean.", Author: "Isaac Newton"},
	{Label: "Quote of the Day", Text: "The best time to plant a tree was 20 years ago. The second best time is now.", Author: "Chinese Proverb"},
	{Label: "Quote of the Day", Text: "Not all those who wander are lost.", Author: "J.R.R. Tolkien", Source: "The Fellowship of the Ring"},
	{Label: "Quote of the Day", Text: "Any sufficiently advanced technology is indistinguishable from magic.", Author: "Arthur C. Clarke"},
}

// fetchQuote returns a random quote from the local pool.
func fetchQuote() *QuoteData {
	if len(quotePool) == 0 {
		return nil
	}
	return &quotePool[rand.Intn(len(quotePool))]
}
