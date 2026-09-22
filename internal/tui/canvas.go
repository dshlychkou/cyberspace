package tui

import (
	"image/color"
	"math"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

// Braille Patterns (U+2800): each cell is a 2×4 dot grid.
//
//	(1) (4)     bits 0x01 0x08
//	(2) (5)          0x02 0x10
//	(3) (6)          0x04 0x20
//	(7) (8)          0x40 0x80
var brailleBit = [4][2]byte{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

const brailleBase = 0x2800

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
			cells[y][x] = cell{ch: " ", fg: colorBg}
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

func (c *canvas) clearRect(x, y, w, h int) {
	for dy := range h {
		for dx := range w {
			c.set(x+dx, y+dy, " ", colorBg)
		}
	}
}

func (c *canvas) drawText(x, y int, text string, fg color.Color) {
	for i, ch := range text {
		c.set(x+i, y, string(ch), fg)
	}
}

// setBrailleDot lights one sub-cell pixel (px,py) in 2×4 braille space.
func (c *canvas) setBrailleDot(px, py int, fg color.Color) {
	if px < 0 || py < 0 {
		return
	}
	cx, cy := px/2, py/4
	if !c.inBounds(cx, cy) {
		return
	}
	lx, ly := px%2, py%4
	bit := brailleBit[ly][lx]

	existing := c.cells[cy][cx]
	var pattern rune = brailleBase
	if r, ok := brailleRune(existing.ch); ok {
		pattern = r
	} else if existing.ch != " " {
		// Don't paint over node labels / legend text.
		return
	}
	pattern |= rune(bit)
	c.cells[cy][cx] = cell{ch: string(pattern), fg: fg}
}

func brailleRune(ch string) (rune, bool) {
	r, size := utf8.DecodeRuneInString(ch)
	if size != len(ch) || r < brailleBase || r > brailleBase+0xFF {
		return 0, false
	}
	return r, true
}

// drawEdge paints a smooth braille line in cell coordinates.
// Endpoints stay clear for node labels. solid=false sparsifies dots.
func (c *canvas) drawEdge(x1, y1, x2, y2 int, fg color.Color, solid bool) {
	// Pixel centers of the two cells.
	px1 := x1*2 + 1
	py1 := y1*4 + 2
	px2 := x2*2 + 1
	py2 := y2*4 + 2

	dx := px2 - px1
	dy := py2 - py1
	steps := max(abs(dx), abs(dy))
	if steps < 2 {
		return
	}

	pad := edgePad(steps)
	fx := float64(dx) / float64(steps)
	fy := float64(dy) / float64(steps)
	stride := 1
	if !solid {
		stride = 3
	}

	for i := pad; i <= steps-pad; i += stride {
		px := px1 + int(math.Round(float64(i)*fx))
		py := py1 + int(math.Round(float64(i)*fy))
		c.setBrailleDot(px, py, fg)
		if solid {
			// Slight thickness for selected links.
			c.setBrailleDot(px, py+1, fg)
		}
	}
}

func edgePad(steps int) int {
	switch {
	case steps < 6:
		return 1
	case steps < 12:
		return 3
	default:
		return 5 // ~1–1.5 cells clear at each end
	}
}

func (c *canvas) render() string {
	var sb strings.Builder
	for y := range c.h {
		var prevFg color.Color
		var run strings.Builder
		for x := range c.w {
			cell := c.cells[y][x]
			if cell.fg != prevFg {
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
