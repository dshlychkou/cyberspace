package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	colorNeonGreen  = lipgloss.Color("#39FF14")
	colorNeonCyan   = lipgloss.Color("#00FFFF")
	colorNeonRed    = lipgloss.Color("#FF073A")
	colorNeonMagenta = lipgloss.Color("#FF00FF")
	colorNeonYellow = lipgloss.Color("#FFE600")
	colorWhite      = lipgloss.Color("#FFFFFF")
	colorDim        = lipgloss.Color("#555555")
	colorBg         = lipgloss.Color("#0A0A0A")
	colorBorder     = lipgloss.Color("#333333")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorNeonCyan).
			Background(colorBg)

	styleProgram = lipgloss.NewStyle().
			Foreground(colorNeonGreen)

	styleICE = lipgloss.NewStyle().
			Foreground(colorNeonRed)

	styleVirus = lipgloss.NewStyle().
			Foreground(colorNeonMagenta)

	styleFirewall = lipgloss.NewStyle().
			Foreground(colorNeonYellow)

	styleCore = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	styleData = lipgloss.NewStyle().
			Foreground(colorNeonCyan)

	styleRelay = lipgloss.NewStyle().
			Foreground(colorDim)

	styleServer = lipgloss.NewStyle().
			Foreground(colorNeonGreen)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorNeonCyan).
			Bold(true)

	styleEdge = lipgloss.NewStyle().
			Foreground(colorDim)

	styleEvent = lipgloss.NewStyle().
			Foreground(colorDim)

	styleHUD = lipgloss.NewStyle().
			Foreground(colorNeonCyan)

	styleThreatLow = lipgloss.NewStyle().
			Foreground(colorNeonGreen)

	styleThreatMed = lipgloss.NewStyle().
			Foreground(colorNeonYellow)

	styleThreatHigh = lipgloss.NewStyle().
			Foreground(colorNeonRed)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorDim)

	styleScore = lipgloss.NewStyle().
			Foreground(colorNeonGreen).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorNeonRed)
)
