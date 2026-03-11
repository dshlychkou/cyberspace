package tui

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/dshlychkou/cyberspace/internal/game"
	"github.com/dshlychkou/cyberspace/internal/network"
)

type nodePos struct {
	x, y int
	id   uint64
}

func layoutNodes(snap *game.StateSnapshot, w, h int) []nodePos {
	cx := w / 2
	cy := h / 2

	// Group nodes by type into layers
	var coreIDs, fwIDs, srvIDs, outerIDs []uint64
	for id, n := range snap.Nodes {
		switch n.Type {
		case network.NodeCore:
			coreIDs = append(coreIDs, id)
		case network.NodeFirewall:
			fwIDs = append(fwIDs, id)
		case network.NodeServer:
			srvIDs = append(srvIDs, id)
		default:
			outerIDs = append(outerIDs, id)
		}
	}
	sort.Slice(coreIDs, func(i, j int) bool { return coreIDs[i] < coreIDs[j] })
	sort.Slice(fwIDs, func(i, j int) bool { return fwIDs[i] < fwIDs[j] })
	sort.Slice(srvIDs, func(i, j int) bool { return srvIDs[i] < srvIDs[j] })
	sort.Slice(outerIDs, func(i, j int) bool { return outerIDs[i] < outerIDs[j] })

	// Compute max radius that stays within bounds after aspect-ratio scaling.
	// Generous margins to account for label width (~8 chars) and entity tags below nodes.
	const marginX = 10 // room for widest label centered on node (e.g. "[★CORE]" = 8 runes)
	const marginY = 4  // room for entity tags below + legend row at bottom
	const stretchX = 1.6
	const stretchY = 0.9
	maxRx := float64(w/2-marginX) / stretchX
	maxRy := float64(h/2-marginY) / stretchY
	maxR := math.Min(maxRx, maxRy)
	if maxR < 3 {
		maxR = 3
	}

	r1 := maxR * 0.30
	r2 := maxR * 0.60
	r3 := maxR * 0.82

	var positions []nodePos

	// Core at center
	for _, id := range coreIDs {
		positions = append(positions, nodePos{x: cx, y: cy, id: id})
	}

	// Firewalls - inner ring
	positions = append(positions, ringLayout(fwIDs, cx, cy, r1, -math.Pi/2, w, h)...)

	// Servers - middle ring
	positions = append(positions, ringLayout(srvIDs, cx, cy, r2, -math.Pi/2+math.Pi/6, w, h)...)

	// Relays + Vaults - outer ring
	positions = append(positions, ringLayout(outerIDs, cx, cy, r3, -math.Pi/2+math.Pi/4, w, h)...)

	return positions
}

func ringLayout(ids []uint64, cx, cy int, radius, startAngle float64, w, h int) []nodePos {
	n := len(ids)
	if n == 0 {
		return nil
	}

	positions := make([]nodePos, n)
	for i, id := range ids {
		angle := startAngle + 2*math.Pi*float64(i)/float64(n)
		x := cx + int(math.Round(radius*math.Cos(angle)*1.6))
		y := cy + int(math.Round(radius*math.Sin(angle)*0.9))
		// Clamp to canvas bounds with generous margins for labels and tags.
		// Left/right: labels are ~8 chars wide, centered on x, so need ~6 chars each side.
		// Top: 1 row margin. Bottom: tag row + 2 legend rows.
		x = clampInt(x, 6, w-7)
		y = clampInt(y, 1, h-5)
		positions[i] = nodePos{x: x, y: y, id: id}
	}
	return positions
}

func drawLegend(c *canvas, startY, w int) {
	type legendEntry struct {
		symbol string
		label  string
		fg     color.Color
	}
	entries := []legendEntry{
		{"★", "Core", colorWhite},
		{"◆", "FW", colorNeonYellow},
		{"◆", "Srv", colorNeonGreen},
		{"◇", "Rly", colorDim},
		{"◆", "Vlt", colorNeonCyan},
		{"|", "", colorBorder},
		{"P", "Prog", colorNeonGreen},
		{"I", "ICE", colorNeonRed},
		{"V", "Virus", colorNeonMagenta},
		{"|", "", colorBorder},
		{"$", "Data", colorNeonCyan},
		{"~", "Compute", colorNeonGreen},
		{"×", "Threat", colorNeonRed},
	}

	x := 1
	row := startY
	for _, e := range entries {
		// Calculate width this entry needs
		needed := len([]rune(e.symbol)) + len(e.label) + 2
		if e.label == "" {
			needed = 2
		}
		// Wrap to next row if it won't fit
		if x+needed > w-1 && x > 1 {
			row++
			x = 1
			if row >= c.h {
				break
			}
		}
		if e.label == "" {
			c.drawText(x, row, e.symbol, e.fg)
			x += 2
		} else {
			c.drawText(x, row, e.symbol, e.fg)
			x += len([]rune(e.symbol))
			c.drawText(x, row, "="+e.label+" ", colorDim)
			x += len(e.label) + 2
		}
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func renderGraph(snap *game.StateSnapshot, selectedID uint64, positions []nodePos, w, h int) string {
	if w < 20 || h < 10 {
		return "Terminal too small"
	}

	// Fall back to computing positions if none provided
	if len(positions) == 0 {
		positions = layoutNodes(snap, w, h)
	}

	posMap := make(map[uint64]nodePos)
	for _, p := range positions {
		posMap[p.id] = p
	}

	c := newCanvas(w, h)

	// Draw edges first
	for _, e := range snap.Edges {
		p1, ok1 := posMap[e.From]
		p2, ok2 := posMap[e.To]
		if !ok1 || !ok2 {
			continue
		}
		edgeColor := colorBorder
		// Highlight edges connected to selected node
		if e.From == selectedID || e.To == selectedID {
			edgeColor = colorDim
		}
		c.drawLine(p1.x, p1.y, p2.x, p2.y, edgeColor)
	}

	// Draw flow pulses on top of edges
	drawFlows(c, snap, posMap)

	// Draw nodes on top
	for _, pos := range positions {
		n := snap.Nodes[pos.id]
		programs, ices, viruses := countEntities(n, snap)
		isSelected := pos.id == selectedID

		drawNode(c, pos.x, pos.y, n, programs, ices, viruses, isSelected)
	}

	// Legend at bottom (start at h-2 so it can wrap to h-1 if needed)
	drawLegend(c, h-2, w)

	return c.render()
}

func drawNode(c *canvas, x, y int, n game.NodeSnapshot, programs, ices, viruses int, selected bool) {
	sym := n.Type.Symbol()
	label := shortLabel(n)

	nodeColor := resolveNodeColor(n.Type, selected, programs, ices)

	// Draw node symbol and label, clamped to canvas bounds
	nodeText := sym + label
	if selected {
		nodeText = "[" + nodeText + "]"
	}
	textLen := len([]rune(nodeText))
	textX := max(clampInt(x-textLen/2, 0, c.w-textLen), 0)
	c.drawText(textX, y, nodeText, nodeColor)

	// Draw entity indicators below (skip if would overlap legend area at h-2)
	if y+1 < c.h-2 {
		drawEntityTags(c, x, y+1, programs, ices, viruses)
	}
}

func resolveNodeColor(t network.NodeType, selected bool, programs, ices int) color.Color {
	nodeColor := nodeColorByType(t)
	if selected {
		nodeColor = colorNeonCyan
	}
	if programs > 0 && ices > 0 {
		nodeColor = colorNeonYellow
	}
	return nodeColor
}

func drawEntityTags(c *canvas, x, y, programs, ices, viruses int) {
	var tags []string
	if programs > 0 {
		tags = append(tags, fmt.Sprintf("%dP", programs))
	}
	if ices > 0 {
		tags = append(tags, fmt.Sprintf("%dI", ices))
	}
	if viruses > 0 {
		tags = append(tags, fmt.Sprintf("%dV", viruses))
	}
	if len(tags) == 0 {
		return
	}

	tagStr := strings.Join(tags, " ")
	tagX := max(clampInt(x-len(tagStr)/2, 0, c.w-len(tagStr)), 0)
	offset := tagX
	for i, tag := range tags {
		tagColor := tagColorForIndex(i, programs, ices)
		c.drawText(offset, y, tag, tagColor)
		offset += len(tag) + 1
	}
}

func tagColorForIndex(i, programs, ices int) color.Color {
	if i == 0 && programs > 0 {
		return colorNeonGreen
	}
	if (i == 0 && ices > 0) || (i == 1 && programs > 0 && ices > 0) {
		return colorNeonRed
	}
	return colorNeonMagenta
}

func shortLabel(n game.NodeSnapshot) string {
	switch n.Type {
	case network.NodeCore:
		return "CORE"
	case network.NodeFirewall:
		return fmt.Sprintf("FW%d", n.ID)
	case network.NodeServer:
		return fmt.Sprintf("S%d", n.ID)
	case network.NodeRelay:
		return fmt.Sprintf("R%d", n.ID)
	case network.NodeVault:
		return fmt.Sprintf("V%d", n.ID)
	default:
		return fmt.Sprintf("?%d", n.ID)
	}
}

func nodeColorByType(t network.NodeType) color.Color {
	switch t {
	case network.NodeCore:
		return colorWhite
	case network.NodeFirewall:
		return colorNeonYellow
	case network.NodeServer:
		return colorNeonGreen
	case network.NodeRelay:
		return colorDim
	case network.NodeVault:
		return colorNeonCyan
	default:
		return colorWhite
	}
}

func renderSelectedDetails(snap *game.StateSnapshot, selectedID uint64) string {
	n, ok := snap.Nodes[selectedID]
	if !ok {
		return ""
	}
	programs, ices, viruses := countEntities(n, snap)

	sym := n.Type.Symbol()

	// Header
	header := styleTitle.Render("NODE: " + sym + " " + n.Label)

	// Entities
	var entities []string
	if programs > 0 {
		entities = append(entities, styleProgram.Render(fmt.Sprintf("%dP", programs)))
	}
	if ices > 0 {
		entities = append(entities, styleICE.Render(fmt.Sprintf("%dI", ices)))
	}
	if viruses > 0 {
		entities = append(entities, styleVirus.Render(fmt.Sprintf("%dV", viruses)))
	}
	entityStr := styleEvent.Render("empty")
	if len(entities) > 0 {
		entityStr = strings.Join(entities, " ")
	}

	// Neighbors
	neighbors := findNeighbors(selectedID, snap)
	neighborStr := styleEvent.Render(strings.Join(neighbors, ", "))

	info := header + "  " + entityStr + "  Links: " + neighborStr

	// Contextual hint
	hint := nodeHint(n, programs, ices, snap)
	if hint != "" {
		info += "\n" + styleEvent.Render(">> ") + hint
	}

	return info
}

func nodeHint(n game.NodeSnapshot, programs, ices int, snap *game.StateSnapshot) string {
	switch n.Type {
	case network.NodeFirewall:
		if ices > 0 && programs == 0 {
			return "ICE here. Deploy a virus (V) on a neighbor or spawn a program (S)."
		}
		if ices > 0 && ices > programs {
			return fmt.Sprintf("%dI vs %dP — outnumbered! Spawn more (S) or virus (V) a neighbor.", ices, programs)
		}
		if programs > 0 {
			return "Firewall held. Programs can't auto-spread from here to CORE — select CORE and press S."
		}
		return "Blocks auto-spread. Press S to manually place a program."
	case network.NodeCore:
		if programs >= snap.CoreWinThreshold {
			return fmt.Sprintf("Holding CORE! %d/%d ticks to win. Defend against ICE.", snap.CoreHoldLen, snap.CoreWinDuration)
		}
		if programs > 0 {
			return fmt.Sprintf("%d/%d programs needed. Press S to spawn more.", programs, snap.CoreWinThreshold)
		}
		return "Target node! Select and press S to place a program."
	case network.NodeVault:
		if programs > 0 {
			return fmt.Sprintf("+%d Data/tick from %d program(s).", programs*5, programs)
		}
		return "Programs here earn Data each tick."
	case network.NodeRelay:
		if programs > 0 {
			return fmt.Sprintf("+%d Compute/tick from %d program(s).", programs*2, programs)
		}
		return "Programs here earn Compute each tick."
	default:
		if programs > 0 {
			return "Programs auto-spread to connected nodes."
		}
		return ""
	}
}

type edgeFlow struct {
	from, to nodePos
	fg       color.Color
	symbol   string
	period   int // ticks per full cycle
	pulses   int // number of pulses on this edge
}

func drawFlows(c *canvas, snap *game.StateSnapshot, posMap map[uint64]nodePos) {
	flows := collectFlows(snap, posMap)
	tick := snap.Tick

	for _, f := range flows {
		for p := range f.pulses {
			// Each pulse offset evenly across the period
			phase := (f.period * p) / f.pulses
			progress := float64((tick+phase)%f.period) / float64(f.period)

			px := f.from.x + int(float64(f.to.x-f.from.x)*progress)
			py := f.from.y + int(float64(f.to.y-f.from.y)*progress)

			// Only draw on edge chars or empty space, don't clobber nodes
			existing := c.get(px, py)
			switch existing.ch {
			case " ", charDot, charHBar, charVBar:
				c.set(px, py, f.symbol, f.fg)
			}
		}
	}
}

type flowCounts struct {
	prog  map[uint64]int
	ice   map[uint64]int
	virus map[uint64]int
}

func collectFlows(snap *game.StateSnapshot, posMap map[uint64]nodePos) []edgeFlow {
	fc := buildFlowCounts(snap)
	var flows []edgeFlow
	for _, e := range snap.Edges {
		fromPos, ok1 := posMap[e.From]
		toPos, ok2 := posMap[e.To]
		if !ok1 || !ok2 {
			continue
		}
		flows = edgeFlows(flows, snap.Nodes[e.From], snap.Nodes[e.To], e.From, e.To, fromPos, toPos, &fc)
	}
	return flows
}

func buildFlowCounts(snap *game.StateSnapshot) flowCounts {
	fc := flowCounts{
		prog:  make(map[uint64]int),
		ice:   make(map[uint64]int),
		virus: make(map[uint64]int),
	}
	for _, p := range snap.Programs {
		fc.prog[p.NodeID]++
	}
	for _, ice := range snap.ICEs {
		fc.ice[ice.NodeID]++
	}
	for _, v := range snap.Viruses {
		fc.virus[v.NodeID]++
	}
	return fc
}

func edgeFlows(
	flows []edgeFlow, fromNode, toNode game.NodeSnapshot,
	fromID, toID uint64, fromPos, toPos nodePos, fc *flowCounts,
) []edgeFlow {
	// Data harvest: vault with programs → $ flowing outward
	if fromNode.Type == network.NodeVault && fc.prog[fromID] > 0 {
		flows = append(flows, edgeFlow{from: fromPos, to: toPos, fg: colorNeonCyan, symbol: "$", period: 8, pulses: 2})
	}
	if toNode.Type == network.NodeVault && fc.prog[toID] > 0 {
		flows = append(flows, edgeFlow{from: toPos, to: fromPos, fg: colorNeonCyan, symbol: "$", period: 8, pulses: 2})
	}
	// Compute harvest: relay with programs → ~ flowing outward
	if fromNode.Type == network.NodeRelay && fc.prog[fromID] > 0 {
		flows = append(flows, edgeFlow{from: fromPos, to: toPos, fg: colorNeonGreen, symbol: "~", period: 7, pulses: 2})
	}
	if toNode.Type == network.NodeRelay && fc.prog[toID] > 0 {
		flows = append(flows, edgeFlow{from: toPos, to: fromPos, fg: colorNeonGreen, symbol: "~", period: 7, pulses: 2})
	}
	// ICE threat
	if fc.ice[fromID] > 0 {
		flows = append(flows, edgeFlow{from: fromPos, to: toPos, fg: colorNeonRed, symbol: "×", period: 5, pulses: 1})
	}
	if fc.ice[toID] > 0 {
		flows = append(flows, edgeFlow{from: toPos, to: fromPos, fg: colorNeonRed, symbol: "×", period: 5, pulses: 1})
	}
	// Virus corruption
	if fc.virus[fromID] > 0 {
		flows = append(flows, edgeFlow{from: fromPos, to: toPos, fg: colorNeonMagenta, symbol: "◈", period: 6, pulses: 1})
	}
	if fc.virus[toID] > 0 {
		flows = append(flows, edgeFlow{from: toPos, to: fromPos, fg: colorNeonMagenta, symbol: "◈", period: 6, pulses: 1})
	}
	// Program network activity
	if fc.prog[fromID] > 0 && fc.prog[toID] > 0 {
		flows = append(flows, edgeFlow{from: fromPos, to: toPos, fg: colorDim, symbol: "•", period: 10, pulses: 1})
	}
	return flows
}

func findNeighbors(nodeID uint64, snap *game.StateSnapshot) []string {
	adj := make(map[uint64]bool)
	for _, e := range snap.Edges {
		if e.From == nodeID {
			adj[e.To] = true
		}
		if e.To == nodeID {
			adj[e.From] = true
		}
	}

	nids := make([]uint64, 0, len(adj))
	for id := range adj {
		nids = append(nids, id)
	}
	sort.Slice(nids, func(i, j int) bool { return nids[i] < nids[j] })

	var names []string
	for _, nid := range nids {
		if n, ok := snap.Nodes[nid]; ok {
			names = append(names, n.Label)
		}
	}
	return names
}
