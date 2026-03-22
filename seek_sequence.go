package applypatch

import textmatch "github.com/amxv/apply-patch-go/internal/textmatch"

func seekSequence(lines []string, pattern []string, start int, eof bool) *int {
	return textmatch.SeekSequence(lines, pattern, start, eof)
}

func seekSequenceWith(lines []string, pattern []string, searchStart int, match func(string, string) bool) *int {
	return textmatch.SeekSequenceWith(lines, pattern, searchStart, match)
}

func normalizeMatch(s string) string {
	return textmatch.NormalizeMatch(s)
}
