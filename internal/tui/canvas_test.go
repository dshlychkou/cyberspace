package tui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBrailleDotMergesInCell(t *testing.T) {
	c := newCanvas(4, 2)
	c.setBrailleDot(0, 0, colorNeonCyan) // bit 1
	c.setBrailleDot(1, 0, colorNeonCyan) // bit 4
	ch := c.get(0, 0).ch
	r, _ := utf8.DecodeRuneInString(ch)
	if r&0x01 == 0 || r&0x08 == 0 {
		t.Fatalf("expected bits 1 and 4 set, got U+%04X %q", r, ch)
	}
}

func TestDrawEdgeUsesBrailleNotASCIISlashes(t *testing.T) {
	c := newCanvas(20, 10)
	c.drawEdge(2, 5, 17, 5, colorNeonCyan, true)

	braille := 0
	for x := 0; x < 20; x++ {
		ch := c.get(x, 5).ch
		if r, ok := brailleRune(ch); ok && r != brailleBase {
			braille++
		}
		if ch == "/" || ch == "\\" || ch == "─" {
			t.Fatalf("unexpected ASCII edge char %q at x=%d", ch, x)
		}
	}
	if braille < 3 {
		t.Fatalf("expected several braille cells along edge, got %d", braille)
	}
	if c.get(2, 5).ch != " " {
		t.Fatalf("start cell should be clear for labels, got %q", c.get(2, 5).ch)
	}
	if c.get(17, 5).ch != " " {
		t.Fatalf("end cell should be clear for labels, got %q", c.get(17, 5).ch)
	}
}

func TestDrawEdgeDoesNotOverwriteText(t *testing.T) {
	c := newCanvas(20, 10)
	c.drawText(8, 5, "NODE", colorWhite)
	c.drawEdge(1, 5, 18, 5, colorDim, true)
	if c.get(8, 5).ch != "N" {
		t.Fatalf("edge overwrote label, got %q", c.get(8, 5).ch)
	}
}

func TestRenderGaugeFill(t *testing.T) {
	g := renderGauge("CORE", 4, 8, 8, &styleScore, "4/8")
	if !strings.Contains(g, "CORE") || !strings.Contains(g, "4/8") {
		t.Fatalf("gauge missing label/detail: %q", g)
	}
}
