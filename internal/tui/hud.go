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

	var line2 string
	for _, l := range hudGaugeLayouts {
		line2 = renderHUDGauges(snap, sparks, l)
		if width <= 0 || lipgloss.Width(line2) <= width {
			break
		}
	}
	if width > 0 {
		line1 = truncateWS(line1, width)
		line2 = truncateWS(line2, width)
		if pad := width - lipgloss.Width(line2); pad > 0 {
			line2 += strings.Repeat(" ", pad)
		}
	}

	return line1 + "\n" + line2
}

const (
	hudHeight = 2

	gaugeBarFull    = 12
	gaugeBarCompact = 8
	gaugeBarMin     = 5

	percentMax    = 100
	threatHighPct = 70
	threatMedPct  = 40
)

type hudGaugeLayout struct {
	barLen, sparkLen int
	detail           bool
}

var hudGaugeLayouts = []hudGaugeLayout{
	{barLen: gaugeBarFull, sparkLen: sparkWidth, detail: true},
	{barLen: gaugeBarCompact, sparkLen: gaugeBarCompact},
	{barLen: gaugeBarMin, sparkLen: gaugeBarMin},
}

func renderHUDGauges(snap *game.StateSnapshot, sparks *sparkHistory, l hudGaugeLayout) string {
	coreTotal := max(snap.CoreWinDuration, 1)
	coreDetail := fmt.Sprintf("%d/%d", snap.CoreHoldLen, coreTotal)
	if l.detail {
		coreDetail += fmt.Sprintf(" ≥%dP", snap.CoreWinThreshold)
	}
	coreGauge := renderGauge("CORE", snap.CoreHoldLen, coreTotal, l.barLen, &styleScore, coreDetail)

	iceCount := len(snap.ICEs)
	progCount := len(snap.Programs)
	threatPct := 0
	if progCount+iceCount > 0 {
		threatPct = (iceCount * percentMax) / (progCount + iceCount)
	}
	threatStyle := styleThreatLow
	switch {
	case threatPct > threatHighPct:
		threatStyle = styleThreatHigh
	case threatPct > threatMedPct:
		threatStyle = styleThreatMed
	}
	threatGauge := renderGauge("Threat", threatPct, percentMax, l.barLen, &threatStyle, fmt.Sprintf("%d%%", threatPct))

	dataNet := snap.DataIncome - snap.DataBurn
	computeNet := snap.ComputeIncome - snap.ComputeBurn
	var dataSpark, computeSpark string
	if sparks != nil {
		dataSpark = renderSparkline(sparks.data, l.sparkLen, &styleScore) + " "
		computeSpark = renderSparkline(sparks.compute, l.sparkLen, &styleSelected) + " "
	}
	dataStr := styleHUD.Render("D ") + dataSpark + formatRate(snap.Resources.Data, dataNet)
	computeStr := styleHUD.Render("C ") + computeSpark + formatRate(snap.Resources.Compute, computeNet)

	return coreGauge + "  " + threatGauge + "  " + dataStr + "  " + computeStr
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
	filled := max(min((current*barLen)/total, barLen), 0)

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
