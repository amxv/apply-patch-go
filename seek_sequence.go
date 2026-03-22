package applypatch

import "strings"

func seekSequence(lines []string, pattern []string, start int, eof bool) *int {
	if len(pattern) == 0 {
		v := start
		return &v
	}
	if len(pattern) > len(lines) {
		return nil
	}
	searchStart := start
	if eof && len(lines) >= len(pattern) {
		searchStart = len(lines) - len(pattern)
	}
	if idx := seekSequenceWith(lines, pattern, searchStart, func(lhs, rhs string) bool { return lhs == rhs }); idx != nil {
		return idx
	}
	if idx := seekSequenceWith(lines, pattern, searchStart, func(lhs, rhs string) bool {
		return strings.TrimRight(lhs, " \t\r\n") == strings.TrimRight(rhs, " \t\r\n")
	}); idx != nil {
		return idx
	}
	if idx := seekSequenceWith(lines, pattern, searchStart, func(lhs, rhs string) bool {
		return strings.TrimSpace(lhs) == strings.TrimSpace(rhs)
	}); idx != nil {
		return idx
	}
	if idx := seekSequenceWith(lines, pattern, searchStart, func(lhs, rhs string) bool {
		return normalizeMatch(lhs) == normalizeMatch(rhs)
	}); idx != nil {
		return idx
	}
	return nil
}

func seekSequenceWith(lines []string, pattern []string, searchStart int, match func(string, string) bool) *int {
	limit := len(lines) - len(pattern)
	for i := searchStart; i <= limit; i++ {
		ok := true
		for j, pat := range pattern {
			if !match(lines[i+j], pat) {
				ok = false
				break
			}
		}
		if ok {
			v := i
			return &v
		}
	}
	return nil
}

func normalizeMatch(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch r {
		case '‐', '‑', '‒', '–', '—', '―', '−':
			b.WriteRune('-')
		case '‘', '’', '‚', '‛':
			b.WriteRune('\'')
		case '“', '”', '„', '‟':
			b.WriteRune('"')
		case '\u00A0', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '　':
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
