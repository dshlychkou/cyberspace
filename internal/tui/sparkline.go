package tui

import (
	"charm.land/lipgloss/v2"
)

const sparkHistoryCap = 24
const sparkWidth = 12

// sparkLevels: braille columns filled from the bottom (0 empty → 4 full).
var sparkLevels = [5]rune{
	brailleBase,        // ⠀
	brailleBase | 0xC0, // ⣀  dots 7+8
	brailleBase | 0xE4, // ⣤  +3+6
	brailleBase | 0xF6, // ⣶  +2+5
	brailleBase | 0xFF, // ⣿
}

type sparkHistory struct {
	data    []int
	compute []int
	tick    int
}

func (h *sparkHistory) reset() {
	h.data = h.data[:0]
	h.compute = h.compute[:0]
	h.tick = -1
}

// push records a sample when the tick advances (or on first sample).
func (h *sparkHistory) push(tick, data, compute int) {
	if tick == h.tick {
		return
	}
	h.tick = tick
	h.data = appendCapped(h.data, data, sparkHistoryCap)
	h.compute = appendCapped(h.compute, compute, sparkHistoryCap)
}

func appendCapped(buf []int, v, capN int) []int {
	buf = append(buf, v)
	if len(buf) > capN {
		copy(buf, buf[len(buf)-capN:])
		buf = buf[:capN]
	}
	return buf
}

// renderSparkline draws a braille bar sparkline scaled to the window max.
func renderSparkline(values []int, width int, style *lipgloss.Style) string {
	if width < 1 {
		return ""
	}
	if len(values) == 0 {
		return style.Render(string(sparkLevels[0]))
	}

	start := 0
	if len(values) > width {
		start = len(values) - width
	}
	window := values[start:]

	maxV := 0
	for _, v := range window {
		if v > maxV {
			maxV = v
		}
	}

	runes := make([]rune, len(window))
	for i, v := range window {
		level := 0
		if maxV > 0 && v > 0 {
			level = (v * 4) / maxV
			if level == 0 {
				level = 1 // show a blip for any positive value
			}
			if level > 4 {
				level = 4
			}
		}
		runes[i] = sparkLevels[level]
	}
	return style.Render(string(runes))
}
