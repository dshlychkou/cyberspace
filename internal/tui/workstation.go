package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/dshlychkou/cyberspace/internal/game"
)

const workstationCap = 48

type consoleKind int

const (
	ckBoot consoleKind = iota
	ckOps
	ckIdle
)

type consoleLine struct {
	kind consoleKind
	text string
}

// workstation is a CRT-style console feed for the right panel.
type workstation struct {
	lines    []consoleLine
	lastEv   int
	lastTick int
	booted   bool
}

func (w *workstation) reset() {
	w.lines = w.lines[:0]
	w.lastEv = 0
	w.lastTick = -1
	w.booted = false
}

func (w *workstation) push(kind consoleKind, text string) {
	w.lines = append(w.lines, consoleLine{kind: kind, text: text})
	if len(w.lines) > workstationCap {
		copy(w.lines, w.lines[len(w.lines)-workstationCap:])
		w.lines = w.lines[:workstationCap]
	}
}

func (w *workstation) sync(snap *game.StateSnapshot) {
	if snap == nil {
		return
	}
	if !w.booted {
		w.push(ckBoot, "INIT NODE-OPS v1.0")
		w.push(ckBoot, "LINK_OK  cipher=neon")
		w.push(ckBoot, "mount /net/graph  rw")
		w.booted = true
	}

	// New game / load can shrink the event list — resync.
	if len(snap.Events) < w.lastEv {
		w.lastEv = 0
	}
	for i := w.lastEv; i < len(snap.Events); i++ {
		e := snap.Events[i]
		w.push(ckOps, fmt.Sprintf("[%d] %s", e.Tick, e.Message))
	}
	w.lastEv = len(snap.Events)

	if snap.Tick != w.lastTick {
		if snap.Tick > 0 && snap.Tick%8 == 0 {
			w.push(ckIdle, idleStatus(snap.Tick))
		}
		w.lastTick = snap.Tick
	}
}

func idleStatus(tick int) string {
	msgs := []string{
		"scan… noise floor nominal",
		"keepalive ping OK",
		"heap chk 0xOK",
		"watchdog armed",
		"bus idle — awaiting ops",
	}
	return msgs[tick%len(msgs)]
}

func renderWorkstation(snap *game.StateSnapshot, selectedID uint64, width, height int, ws *workstation) string {
	if width < 8 {
		width = 8
	}
	if height < 6 {
		height = 6
	}

	title := styleTitle.Render("SYS://NODE-OPS")
	obj := renderWSObjective(snap)
	keys := renderWSKeys(snap)
	lock := renderWSLock(snap, selectedID)
	probe := renderWSProbe(snap, selectedID)
	pulse := renderWSPulse(snap)

	header := []string{title, obj, keys, "", lock, probe, styleEvent.Render(strings.Repeat("─", min(width, 22))), ""}
	footer := []string{"", pulse, renderWSCursor(snap.Tick)}

	headerH := len(header)
	footerH := len(footer)
	consoleH := height - headerH - footerH
	if consoleH < 2 {
		consoleH = 2
	}

	console := renderWSConsole(ws, width, consoleH)

	var sb strings.Builder
	for _, line := range header {
		sb.WriteString(truncateWS(line, width))
		sb.WriteByte('\n')
	}
	sb.WriteString(console)
	for _, line := range footer {
		sb.WriteByte('\n')
		sb.WriteString(truncateWS(line, width))
	}
	return sb.String()
}

func renderWSObjective(snap *game.StateSnapshot) string {
	return styleEvent.Render("OBJ ") +
		styleProgram.Render(fmt.Sprintf("%d+P", snap.CoreWinThreshold)) +
		styleEvent.Render(" on ") +
		styleCore.Render("★CORE") +
		styleEvent.Render(fmt.Sprintf(" ×%d", snap.CoreWinDuration))
}

func renderWSKeys(snap *game.StateSnapshot) string {
	return styleEvent.Render("KEY ") +
		styleSelected.Render("←↑↓→") + " " +
		styleSelected.Render("S") + styleEvent.Render(fmt.Sprintf("-%dD ", snap.ProgramSpawnCost)) +
		styleSelected.Render("V") + styleEvent.Render(fmt.Sprintf("-%dC ", snap.VirusDeployCost)) +
		styleSelected.Render("Spc")
}

func renderWSLock(snap *game.StateSnapshot, selectedID uint64) string {
	n, ok := snap.Nodes[selectedID]
	if !ok {
		return styleEvent.Render("> LOCK ") + styleError.Render("— none —")
	}
	p, i, v := countEntities(n, snap)
	return styleEvent.Render("> LOCK ") +
		styleData.Render(n.Type.Symbol()+" "+n.Label) + " " +
		styleProgram.Render(fmt.Sprintf("%dP", p)) + " " +
		styleICE.Render(fmt.Sprintf("%dI", i)) + " " +
		styleVirus.Render(fmt.Sprintf("%dV", v))
}

func renderWSProbe(snap *game.StateSnapshot, selectedID uint64) string {
	neighbors := findNeighbors(selectedID, snap)
	if len(neighbors) == 0 {
		return styleEvent.Render("> PROBE ") + styleEvent.Render("(no links)")
	}
	maxN := 3
	shown := neighbors
	extra := ""
	if len(shown) > maxN {
		extra = fmt.Sprintf(" +%d", len(shown)-maxN)
		shown = shown[:maxN]
	}
	return styleEvent.Render("> PROBE ") + styleHUD.Render(strings.Join(shown, ",")) + styleEvent.Render(extra)
}

func renderWSPulse(snap *game.StateSnapshot) string {
	dNet := snap.DataIncome - snap.DataBurn
	cNet := snap.ComputeIncome - snap.ComputeBurn
	return styleEvent.Render("> PULSE ") +
		styleData.Render(fmt.Sprintf("D%d", snap.Resources.Data)) + rateTag(dNet) + " " +
		styleSelected.Render(fmt.Sprintf("C%d", snap.Resources.Compute)) + rateTag(cNet)
}

func rateTag(net int) string {
	if net > 0 {
		return styleScore.Render(fmt.Sprintf("+%d", net))
	}
	if net < 0 {
		return styleError.Render(fmt.Sprintf("%d", net))
	}
	return styleEvent.Render("+0")
}

func renderWSCursor(tick int) string {
	if tick%2 == 0 {
		return styleSelected.Render("▌")
	}
	return styleEvent.Render("▌")
}

func renderWSConsole(ws *workstation, width, height int) string {
	if ws == nil || len(ws.lines) == 0 {
		return styleEvent.Render("> _")
	}
	start := 0
	if len(ws.lines) > height {
		start = len(ws.lines) - height
	}
	var sb strings.Builder
	for i, line := range ws.lines[start:] {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(truncateWS(styleConsoleLine(line), width))
	}
	// Pad to height so the CRT area feels solid.
	shown := len(ws.lines) - start
	for shown < height {
		sb.WriteByte('\n')
		sb.WriteString(styleEvent.Render(strings.Repeat(" ", min(width, 1))))
		shown++
	}
	return sb.String()
}

func styleConsoleLine(line consoleLine) string {
	prompt := styleHUD.Render("> ")
	switch line.kind {
	case ckBoot:
		return prompt + styleEvent.Render(line.text)
	case ckIdle:
		return prompt + styleEvent.Render(line.text)
	default:
		style := styleProgram
		lower := strings.ToLower(line.text)
		if strings.Contains(lower, "die") || strings.Contains(lower, "fail") ||
			strings.Contains(lower, "over") || strings.Contains(lower, "ice") {
			style = styleICE
		}
		return prompt + style.Render(line.text)
	}
}

func truncateWS(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

// renderSidebar is the right-panel entry point (CRT workstation).
func renderSidebar(snap *game.StateSnapshot, selectedID uint64, width, height int, ws *workstation) string {
	return renderWorkstation(snap, selectedID, width, height, ws)
}
