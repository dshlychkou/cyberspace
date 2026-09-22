package tui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSparkHistoryPushAdvancesOnTick(t *testing.T) {
	var h sparkHistory
	h.reset()
	h.push(0, 10, 5)
	h.push(0, 99, 99) // same tick — ignored
	h.push(1, 12, 6)
	if len(h.data) != 2 || h.data[0] != 10 || h.data[1] != 12 {
		t.Fatalf("data hist = %v", h.data)
	}
	if len(h.compute) != 2 || h.compute[1] != 6 {
		t.Fatalf("compute hist = %v", h.compute)
	}
}

func TestSparkHistoryCaps(t *testing.T) {
	var h sparkHistory
	h.reset()
	for i := range sparkHistoryCap + 5 {
		h.push(i, i, i)
	}
	if len(h.data) != sparkHistoryCap {
		t.Fatalf("want cap %d, got %d", sparkHistoryCap, len(h.data))
	}
	if h.data[0] != 5 || h.data[len(h.data)-1] != sparkHistoryCap+4 {
		t.Fatalf("unexpected window: first=%d last=%d", h.data[0], h.data[len(h.data)-1])
	}
}

func TestRenderSparklineScales(t *testing.T) {
	s := renderSparkline([]int{0, 50, 100}, 3, &styleScore)
	// Strip ANSI to count runes — lipgloss may add codes; check contains full/empty braille.
	if !strings.Contains(s, string(sparkLevels[4])) {
		t.Fatalf("expected full braille for max, got %q", s)
	}
	if !strings.Contains(s, string(sparkLevels[0])) {
		t.Fatalf("expected empty braille for zero, got %q", s)
	}
}

func TestRenderSparklineWidthTrim(t *testing.T) {
	vals := make([]int, 20)
	for i := range vals {
		vals[i] = i + 1
	}
	s := renderSparkline(vals, 8, &styleEvent)
	plain := stripANSI(s)
	if utf8.RuneCountInString(plain) != 8 {
		t.Fatalf("want 8 spark cells, got %d (%q)", utf8.RuneCountInString(plain), plain)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc:
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
