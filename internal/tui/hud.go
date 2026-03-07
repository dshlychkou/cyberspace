package tui

import (
	"fmt"
	"strings"

	"github.com/dshlychkou/cyberspace/internal/game"
)

func renderHUD(snap game.StateSnapshot, width int) string {
	title := styleTitle.Render("CYBERSPACE")

	tickStr := styleHUD.Render(fmt.Sprintf("Tick:%d", snap.Tick))

	// Programs / ICE counts
	progStr := styleProgram.Render(fmt.Sprintf("%dP", len(snap.Programs)))
	iceStr := styleICE.Render(fmt.Sprintf("%dI", len(snap.ICEs)))

	// Status
	statusStr := ""
	if snap.Paused {
		if snap.Tick == 0 {
			statusStr = styleSelected.Render(" [SPACE TO START]")
		} else {
			statusStr = styleError.Render(" [PAUSED]")
		}
	}
	if snap.GameOver {
		if snap.Won {
			statusStr = styleScore.Render(" [YOU WIN!]")
		} else {
			statusStr = styleError.Render(" [GAME OVER]")
		}
	}

	// Core hold progress — the central mechanic, make it visible
	coreStr := ""
	if snap.CoreHoldLen > 0 {
		held := snap.CoreHoldLen
		total := snap.CoreWinDuration
		bar := renderProgressBar(held, total, 10)
		coreStr = styleScore.Render(fmt.Sprintf(" CORE[%s %d/%d]", bar, held, total))
	}

	left := title + " " + tickStr + " " + progStr + " " + iceStr + statusStr + coreStr

	// Threat bar on right
	iceCount := len(snap.ICEs)
	progCount := len(snap.Programs)
	threatPct := 0
	if progCount+iceCount > 0 {
		threatPct = (iceCount * 100) / (progCount + iceCount)
	}
	threatBar := renderThreatBar(threatPct)
	right := fmt.Sprintf("Threat:%s%d%%", threatBar, threatPct)

	gap := width - len(stripAnsi(left)) - len(stripAnsi(right))
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func renderProgressBar(current, total, barLen int) string {
	filled := (current * barLen) / total
	if filled > barLen {
		filled = barLen
	}
	var bar strings.Builder
	for i := range barLen {
		if i < filled {
			bar.WriteString(styleScore.Render("█"))
		} else {
			bar.WriteString(styleEvent.Render("░"))
		}
	}
	return bar.String()
}

func renderThreatBar(pct int) string {
	total := 8
	filled := (pct * total) / 100
	if filled > total {
		filled = total
	}

	var bar strings.Builder
	for i := range total {
		if i < filled {
			if pct > 70 {
				bar.WriteString(styleThreatHigh.Render("█"))
			} else if pct > 40 {
				bar.WriteString(styleThreatMed.Render("█"))
			} else {
				bar.WriteString(styleThreatLow.Render("█"))
			}
		} else {
			bar.WriteString(styleEvent.Render("░"))
		}
	}
	return bar.String()
}

func stripAnsi(s string) string {
	result := make([]byte, 0, len(s))
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		result = append(result, s[i])
	}
	return string(result)
}
