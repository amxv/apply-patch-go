package applypatch

import "testing"

func TestSeekSequenceExact(t *testing.T) {
	lines := []string{"foo", "bar", "baz"}
	pattern := []string{"bar", "baz"}
	idx := seekSequence(lines, pattern, 0, false)
	if idx == nil || *idx != 1 {
		t.Fatalf("unexpected index: %v", idx)
	}
}

func TestSeekSequenceTrimmed(t *testing.T) {
	lines := []string{"  foo  ", "   bar\t"}
	pattern := []string{"foo", "bar"}
	idx := seekSequence(lines, pattern, 0, false)
	if idx == nil || *idx != 0 {
		t.Fatalf("unexpected index: %v", idx)
	}
}

func TestSeekSequenceUnicodeNormalized(t *testing.T) {
	lines := []string{"import asyncio  # local import – avoids top‑level dep"}
	pattern := []string{"import asyncio  # local import - avoids top-level dep"}
	idx := seekSequence(lines, pattern, 0, false)
	if idx == nil || *idx != 0 {
		t.Fatalf("unexpected index: %v", idx)
	}
}
