package textx

import (
	"strings"
	"time"
)

const (
	// wpm is the words per minute average for English.
	//
	// https://en.wikipedia.org/wiki/Words_per_minute#Reading_and_comprehension
	wpm int = 228
)

// CountWords performs a rough count of the words in the given string.
func CountWords(text string) int {
	return len(strings.Fields(text))
}

func ReadingTime(text string) time.Duration {
	words := len(strings.Fields(text))
	minutes := float64(words) / float64(wpm)
	return time.Duration(minutes * float64(time.Minute))
}
