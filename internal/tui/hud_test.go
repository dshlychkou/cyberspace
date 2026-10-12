package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/dshlychkou/cyberspace/v2/internal/game"
)

func TestRenderHUDFitsWidth(t *testing.T) {
	snap := &game.StateSnapshot{Paused: true}
	var h sparkHistory
	h.reset()
	for i := range sparkHistoryCap + 1 {
		h.push(i, i, i)
	}

	var widths []int
	for _, l := range hudGaugeLayouts {
		w := lipgloss.Width(renderHUDGauges(snap, &h, l))
		widths = append(widths, w, w-1)
	}

	for _, width := range widths {
		out := renderHUD(snap, width, time.Second, &h)
		lines := strings.Split(out, "\n")
		if len(lines) != hudHeight {
			t.Fatalf("width %d: want %d lines, got %d", width, hudHeight, len(lines))
		}
		for i, l := range lines {
			if w := lipgloss.Width(l); w > width {
				t.Fatalf("width %d: line %d is %d cols wide", width, i, w)
			}
		}
	}
}
