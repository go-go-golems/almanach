package app

import (
	"time"
)

// fetchDate returns the current date as a DateData block.
// This is a local computation — no network request needed.
func fetchDate(t time.Time) *DateData {
	return &DateData{
		Date: t.Format("2006-01-02"),
		Day:  t.Format("Monday"),
	}
}
