package applypatch

import "testing"

func TestSeekSequenceAndNormalizeHelpers(t *testing.T) {
	idx := seekSequence([]string{"a", "b"}, []string{}, 1, false)
	if idx == nil || *idx != 1 {
		t.Fatalf("unexpected empty-pattern index: %#v", idx)
	}

	idx = seekSequence([]string{"one", "two", "three"}, []string{"three"}, 0, true)
	if idx == nil || *idx != 2 {
		t.Fatalf("unexpected eof index: %#v", idx)
	}

	if got := normalizeMatch("‘quoted’"); got != "'quoted'" {
		t.Fatalf("unexpected quote normalization: %q", got)
	}
	if got := normalizeMatch("x\u00A0y"); got != "x y" {
		t.Fatalf("unexpected space normalization: %q", got)
	}
}
