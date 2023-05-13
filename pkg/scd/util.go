package scd

import "fmt"

func Colorize(color, text string) string {
	// Map color names to ANSI escape codes
	colors := map[string]string{
		"black":   "30",
		"red":     "31",
		"green":   "32",
		"yellow":  "33",
		"blue":    "34",
		"magenta": "35",
		"cyan":    "36",
		"white":   "37",
	}

	// Check if the specified color is valid
	code, ok := colors[color]
	if !ok {
		return text // Invalid color, return unmodified text
	}

	// Return formatted text with the specified color
	return fmt.Sprintf("\033[%sm%s\033[0m", code, text)
}

func createChunks[T any](slice *[]T, size int) [][]T {
	chunks := [][]T{}
	for i := 0; i < len(*slice); i += size {
		end := i + size
		if end > len(*slice) {
			end = len(*slice)
		}
		chunks = append(chunks, (*slice)[i:end])
	}
	return chunks
}
