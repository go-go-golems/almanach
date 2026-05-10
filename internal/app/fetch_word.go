package app

import (
	"math/rand"
)

var wordPool = []WordData{
	{Label: "Word of the Day", Word: "serendipity", Definition: "The occurrence of events by chance in a happy or beneficial way.", Part: "noun", Example: "A fortunate stroke of serendipity."},
	{Label: "Word of the Day", Word: "ephemeral", Definition: "Lasting for a very short time.", Part: "adjective", Example: "Fashions are ephemeral: new ones regularly emerge."},
	{Label: "Word of the Day", Word: "petrichor", Definition: "A pleasant smell that frequently accompanies the first rain after a long period of warm, dry weather.", Part: "noun"},
	{Label: "Word of the Day", Word: "mellifluous", Definition: "A sound that is pleasingly smooth and musical to hear.", Part: "adjective"},
	{Label: "Word of the Day", Word: "lacuna", Definition: "An unfilled space or gap; a missing part.", Part: "noun", Example: "There are lacunae in our knowledge of the period."},
	{Label: "Word of the Day", Word: "syzygy", Definition: "A roughly straight-line configuration of three or more celestial bodies in a gravitational system.", Part: "noun"},
	{Label: "Word of the Day", Word: "apricity", Definition: "The warmth of the sun in winter.", Part: "noun"},
}

// fetchWord returns a random word from the local pool.
func fetchWord() *WordData {
	if len(wordPool) == 0 {
		return nil
	}
	return &wordPool[rand.Intn(len(wordPool))]
}
