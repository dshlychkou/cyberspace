package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/dshlychkou/cyberspace/v2/internal/game"
)

func renderHUD(snap *game.StateSnapshot, width int, tickRate time.Duration, sparks *sparkHistory) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(colorNeonPurple).Render("CYBERSPACE")

	tickStr := styleHUD.Render(fmt.Sprintf("Tick:%d", snap.Tick))
	speedStr := styleEvent.Render(fmt.Sprintf(" %dms", tickRate.Milliseconds()))

	progStr := styleProgram.Render(fmt.Sprintf("%dP", len(snap.Programs)))
	iceStr := styleICE.Render(fmt.Sprintf("%dI", len(snap.ICEs)))

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
			statusStr = styleScore.Render(" [YOU WIN! R=NEW GAME]")
		} else {
			statusStr = styleError.Render(" [GAME OVER R=NEW GAME]")
		}
	}

	line1 := title + " " + tickStr + speedStr + " " + progStr + " " + iceStr + statusStr

	coreTotal := snap.CoreWinDuration
	if coreTotal < 1 {
		coreTotal = 1
	}
	coreGauge := renderGauge(
		"CORE",
		snap.CoreHoldLen,
		coreTotal,
		12,
		&styleScore,
		fmt.Sprintf("%d/%d ≥%dP", snap.CoreHoldLen, coreTotal, snap.CoreWinThreshold),
	)

	iceCount := len(snap.ICEs)
	progCount := len(snap.Programs)
	threatPct := 0
	if progCount+iceCount > 0 {
		threatPct = (iceCount * 100) / (progCount + iceCount)
	}
	threatStyle := styleThreatLow
	switch {
	case threatPct > 70:
		threatStyle = styleThreatHigh
	case threatPct > 40:
		threatStyle = styleThreatMed
	}
	threatGauge := renderGauge("Threat", threatPct, 100, 12, &threatStyle, fmt.Sprintf("%d%%", threatPct))

	dataNet := snap.DataIncome - snap.DataBurn
	computeNet := snap.ComputeIncome - snap.ComputeBurn
	var dataSpark, computeSpark string
	if sparks != nil {
		dataSpark = renderSparkline(sparks.data, sparkWidth, &styleScore)
		computeSpark = renderSparkline(sparks.compute, sparkWidth, &styleSelected)
	}
	dataStr := styleHUD.Render("D ") + dataSpark + " " + formatRate(snap.Resources.Data, dataNet)
	computeStr := styleHUD.Render("C ") + computeSpark + " " + formatRate(snap.Resources.Compute, computeNet)

	line2 := coreGauge + "  " + threatGauge + "  " + dataStr + "  " + computeStr
	if width > 0 {
		pad := width - lipgloss.Width(line2)
		if pad > 0 {
			line2 += strings.Repeat(" ", pad)
		}
	}

	return line1 + "\n" + line2
}

func formatRate(current, net int) string {
	rateStr := ""
	if net > 0 {
		rateStr = styleScore.Render(fmt.Sprintf("+%d", net))
	} else if net < 0 {
		rateStr = styleError.Render(fmt.Sprintf("%d", net))
	} else {
		rateStr = styleEvent.Render("+0")
	}
	return styleHUD.Render(fmt.Sprintf("%d", current)) + rateStr
}

// renderGauge draws a btop-style bordered bar: LABEL │████░░░░│ detail
func renderGauge(label string, current, total, barLen int, fillStyle *lipgloss.Style, detail string) string {
	if total < 1 {
		total = 1
	}
	filled := (current * barLen) / total
	if filled > barLen {
		filled = barLen
	}
	if filled < 0 {
		filled = 0
	}

	var bar strings.Builder
	bar.WriteString(styleEvent.Render("│"))
	for i := range barLen {
		if i < filled {
			bar.WriteString(fillStyle.Render("█"))
		} else {
			bar.WriteString(styleEvent.Render("░"))
		}
	}
	bar.WriteString(styleEvent.Render("│"))

	return styleHUD.Render(label+" ") + bar.String() + styleEvent.Render(" "+detail)
}
