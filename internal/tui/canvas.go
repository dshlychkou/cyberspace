package tui

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	charDot  = "·"
	charHBar = "─"
	charVBar = "│"
)

type cell struct {
	ch string
	fg color.Color
}

type canvas struct {
	cells [][]cell
	w, h  int
}

func newCanvas(w, h int) *canvas {
	cells := make([][]cell, h)
	for y := range h {
		cells[y] = make([]cell, w)
		for x := range w {
			// Subtle grid dots for CRT/matrix vibe
			if x%4 == 0 && y%2 == 0 {
				cells[y][x] = cell{ch: charDot, fg: colorGridDot}
			} else {
				cells[y][x] = cell{ch: " ", fg: colorBg}
			}
		}
	}
	return &canvas{cells: cells, w: w, h: h}
}

func (c *canvas) inBounds(x, y int) bool {
	return x >= 0 && x < c.w && y >= 0 && y < c.h
}

func (c *canvas) set(x, y int, ch string, fg color.Color) {
	if c.inBounds(x, y) {
		c.cells[y][x] = cell{ch: ch, fg: fg}
	}
}

func (c *canvas) get(x, y int) cell {
	if c.inBounds(x, y) {
		return c.cells[y][x]
	}
	return cell{ch: " ", fg: colorBg}
}

func (c *canvas) drawText(x, y int, text string, fg color.Color) {
	for i, ch := range text {
		c.set(x+i, y, string(ch), fg)
	}
}

func (c *canvas) drawLine(x1, y1, x2, y2 int, fg color.Color) {
	dx := x2 - x1
	dy := y2 - y1
	steps := max(abs(dx), abs(dy))
	if steps == 0 {
		return
	}

	fx := float64(dx) / float64(steps)
	fy := float64(dy) / float64(steps)

	for i := 1; i < steps; i++ {
		x := x1 + int(math.Round(float64(i)*fx))
		y := y1 + int(math.Round(float64(i)*fy))

		if !c.inBounds(x, y) {
			continue
		}
		// Don't overwrite nodes
		existing := c.get(x, y)
		if existing.ch != " " && existing.ch != charDot && existing.ch != charHBar && existing.ch != charVBar {
			continue
		}

		ch := edgeChar(fx, fy)
		c.set(x, y, ch, fg)
	}
}

func edgeChar(fx, fy float64) string {
	ax := math.Abs(fx)
	ay := math.Abs(fy)

	if ay < 0.15 {
		return charHBar
	}
	if ax < 0.3 {
		return charVBar
	}
	return charDot
}

func (c *canvas) render() string {
	var sb strings.Builder
	for y := range c.h {
		var prevFg color.Color
		var run strings.Builder
		for x := range c.w {
			cell := c.cells[y][x]
			if cell.fg != prevFg {
				// Flush previous run
				if run.Len() > 0 {
					sb.WriteString(lipgloss.NewStyle().Foreground(prevFg).Render(run.String()))
					run.Reset()
				}
				prevFg = cell.fg
			}
			run.WriteString(cell.ch)
		}
		if run.Len() > 0 {
			sb.WriteString(lipgloss.NewStyle().Foreground(prevFg).Render(run.String()))
		}
		if y < c.h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
