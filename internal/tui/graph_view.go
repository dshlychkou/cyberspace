package tui

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/dshlychkou/cyberspace/internal/game"
	"github.com/dshlychkou/cyberspace/internal/network"
)

type nodePos struct {
	x, y int
	id   uint64
}

func layoutNodes(snap *game.StateSnapshot, w, h int, cam camera) []nodePos {
	cx, cy := w/2, h/2
	ids := groupNodeIDs(snap)

	// Leave room for labels (~8 chars) and entity tags under nodes.
	const (
		marginX  = 11
		marginY  = 4
		stretchX = 1.75
		stretchY = 0.88
	)

	maxRx := float64(w/2-marginX) / stretchX
	maxRy := float64(h/2-marginY) / stretchY
	maxR := math.Min(maxRx, maxRy)
	if maxR < 5 {
		maxR = 5
	}

	// Wider ring gaps → less edge crossings through the CORE.
	r1, r2, r3 := maxR*0.28, maxR*0.55, maxR*0.82

	var positions []nodePos
	for _, id := range ids.core {
		positions = append(positions, nodePos{x: cx, y: cy, id: id})
	}
	positions = append(positions, ringLayout(ids.fw, cx, cy, r1, -math.Pi/2+cam.yaw, stretchX, stretchY, w, h)...)
	positions = append(positions, ringLayout(ids.srv, cx, cy, r2, -math.Pi/2+math.Pi/5+cam.yaw, stretchX, stretchY, w, h)...)
	positions = append(positions, ringLayout(ids.outer, cx, cy, r3, -math.Pi/2+math.Pi/7+cam.yaw, stretchX, stretchY, w, h)...)
	return positions
}

type nodeIDGroups struct {
	core, fw, srv, outer []uint64
}

func groupNodeIDs(snap *game.StateSnapshot) nodeIDGroups {
	var g nodeIDGroups
	for id, n := range snap.Nodes {
		switch n.Type {
		case network.NodeCore:
			g.core = append(g.core, id)
		case network.NodeFirewall:
			g.fw = append(g.fw, id)
		case network.NodeServer:
			g.srv = append(g.srv, id)
		default:
			g.outer = append(g.outer, id)
		}
	}
	slices.Sort(g.core)
	slices.Sort(g.fw)
	slices.Sort(g.srv)
	slices.Sort(g.outer)
	return g
}

func ringLayout(ids []uint64, cx, cy int, radius, startAngle, stretchX, stretchY float64, w, h int) []nodePos {
	n := len(ids)
	if n == 0 {
		return nil
	}
	out := make([]nodePos, n)
	for i, id := range ids {
		angle := startAngle + 2*math.Pi*float64(i)/float64(n)
		x := cx + int(math.Round(radius*math.Cos(angle)*stretchX))
		y := cy + int(math.Round(radius*math.Sin(angle)*stretchY))
		x = clampInt(x, 7, w-8)
		y = clampInt(y, 1, h-4)
		out[i] = nodePos{x: x, y: y, id: id}
	}
	return out
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
	if len(positions) == 0 {
		var cam camera
		cam.reset()
		positions = layoutNodes(snap, w, h, cam)
	}

	posMap := make(map[uint64]nodePos, len(positions))
	for _, p := range positions {
		posMap[p.id] = p
	}

	c := newCanvas(w, h)
	drawGraphEdges(c, snap, posMap, selectedID)
	drawGraphNodes(c, snap, positions, selectedID, neighborSet(selectedID, snap))
	drawLegend(c, h-2, w)
	return c.render()
}

func drawGraphEdges(c *canvas, snap *game.StateSnapshot, posMap map[uint64]nodePos, selectedID uint64) {
	// Background topology: sparse dots — shows structure without slash mesh.
	for _, e := range snap.Edges {
		if e.From == selectedID || e.To == selectedID {
			continue
		}
		p1, ok1 := posMap[e.From]
		p2, ok2 := posMap[e.To]
		if !ok1 || !ok2 {
			continue
		}
		c.drawEdge(p1.x, p1.y, p2.x, p2.y, colorDim, false)
	}
	// Selected links: solid bright lines (the ones that matter for play).
	for _, e := range snap.Edges {
		if e.From != selectedID && e.To != selectedID {
			continue
		}
		p1, ok1 := posMap[e.From]
		p2, ok2 := posMap[e.To]
		if !ok1 || !ok2 {
			continue
		}
		c.drawEdge(p1.x, p1.y, p2.x, p2.y, colorNeonCyan, true)
	}
}

func drawGraphNodes(c *canvas, snap *game.StateSnapshot, positions []nodePos, selectedID uint64, neighbors map[uint64]bool) {
	for _, pos := range positions {
		n := snap.Nodes[pos.id]
		programs, ices, viruses := countEntities(n, snap)
		role := nodeDrawRoleNormal
		switch {
		case pos.id == selectedID:
			role = nodeDrawRoleSelected
		case neighbors[pos.id]:
			role = nodeDrawRoleNeighbor
		}
		drawNode(c, pos, n, programs, ices, viruses, role)
	}
}

func neighborSet(selectedID uint64, snap *game.StateSnapshot) map[uint64]bool {
	out := make(map[uint64]bool)
	if selectedID == 0 {
		return out
	}
	for _, e := range snap.Edges {
		if e.From == selectedID {
			out[e.To] = true
		}
		if e.To == selectedID {
			out[e.From] = true
		}
	}
	return out
}

type nodeDrawRole int

const (
	nodeDrawRoleNormal nodeDrawRole = iota
	nodeDrawRoleSelected
	nodeDrawRoleNeighbor
)

func drawLegend(c *canvas, startY, w int) {
	type legendEntry struct {
		symbol, label string
		fg            color.Color
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
	}
	x, row := 1, startY
	for _, e := range entries {
		needed := len([]rune(e.symbol)) + len(e.label) + 2
		if e.label == "" {
			needed = 2
		}
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
			continue
		}
		c.drawText(x, row, e.symbol, e.fg)
		x += len([]rune(e.symbol))
		c.drawText(x, row, "="+e.label+" ", colorDim)
		x += len(e.label) + 2
	}
}

func drawNode(c *canvas, pos nodePos, n game.NodeSnapshot, programs, ices, viruses int, role nodeDrawRole) {
	sym := n.Type.Symbol()
	label := shortLabel(n)
	nodeColor := resolveNodeColor(n.Type, role == nodeDrawRoleSelected, programs, ices)

	nodeText := sym + label
	switch role {
	case nodeDrawRoleSelected:
		nodeText = "[" + nodeText + "]"
		nodeColor = colorNeonCyan
	case nodeDrawRoleNeighbor:
		nodeText = "(" + nodeText + ")"
		if programs == 0 || ices == 0 {
			nodeColor = colorNeonPink
		}
	}

	textLen := len([]rune(nodeText))
	textX := max(clampInt(pos.x-textLen/2, 0, c.w-textLen), 0)

	// Wipe edge crumbs under the label + tag row so nodes own their footprint.
	tagH := 1
	if programs > 0 || ices > 0 || viruses > 0 {
		tagH = 2
	}
	c.clearRect(textX, pos.y, textLen, tagH)

	c.drawText(textX, pos.y, nodeText, nodeColor)
	if pos.y+1 < c.h-2 {
		drawEntityTags(c, pos.x, pos.y+1, programs, ices, viruses)
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
	header := styleTitle.Render("NODE: " + sym + " " + n.Label)

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

	neighbors := findNeighbors(selectedID, snap)
	neighborStr := styleEvent.Render(strings.Join(neighbors, ", "))
	if len(neighbors) == 0 {
		neighborStr = styleEvent.Render("(none)")
	}

	info := header + "  " + entityStr + "  Links: " + neighborStr
	hint := nodeHint(n, programs, ices, snap)
	if hint != "" {
		info += "\n" + styleEvent.Render(">> ") + hint
	}
	return info
}

func nodeHint(n game.NodeSnapshot, programs, ices int, snap *game.StateSnapshot) string {
	costHint := formatCostHint(snap)
	tactical := tacticalHint(n, programs, ices, snap)
	if tactical == "" {
		return costHint
	}
	return tactical + "  |  " + costHint
}

func formatCostHint(snap *game.StateSnapshot) string {
	costHint := fmt.Sprintf("S: −%d Data (%d) · V: −%d Compute (%d)",
		snap.ProgramSpawnCost, snap.Resources.Data,
		snap.VirusDeployCost, snap.Resources.Compute)
	switch {
	case snap.Resources.Data < snap.ProgramSpawnCost:
		return styleError.Render("Can't afford S") + " · " + costHint
	case snap.Resources.Compute < snap.VirusDeployCost:
		return costHint + " · " + styleError.Render("Can't afford V")
	default:
		return costHint
	}
}

func tacticalHint(n game.NodeSnapshot, programs, ices int, snap *game.StateSnapshot) string {
	switch n.Type {
	case network.NodeFirewall:
		switch {
		case ices > 0 && programs == 0:
			return "ICE here. Deploy a virus (V) on a neighbor or spawn a program (S)."
		case ices > 0 && ices > programs:
			return fmt.Sprintf("%dI vs %dP — outnumbered! Spawn more (S) or virus (V) a neighbor.", ices, programs)
		case programs > 0:
			return "Firewall held. Programs can't auto-spread from here to CORE — select CORE and press S."
		default:
			return "Blocks auto-spread. Press S to manually place a program."
		}
	case network.NodeCore:
		switch {
		case programs >= snap.CoreWinThreshold:
			return fmt.Sprintf("Holding CORE! %d/%d ticks to win. Defend against ICE.", snap.CoreHoldLen, snap.CoreWinDuration)
		case programs > 0:
			return fmt.Sprintf("%d/%d programs needed. Press S to spawn more.", programs, snap.CoreWinThreshold)
		default:
			return "Target node! Select and press S to place a program."
		}
	case network.NodeVault:
		if programs > 0 {
			return fmt.Sprintf("+%d Data/tick from %d program(s).", programs*snap.DataHarvestRate, programs)
		}
		return "Programs here earn Data each tick."
	case network.NodeRelay:
		if programs > 0 {
			return fmt.Sprintf("+%d Compute/tick from %d program(s).", programs*snap.ComputeHarvestRate, programs)
		}
		return "Programs here earn Compute each tick."
	default:
		if programs > 0 {
			return "Programs auto-spread to connected nodes."
		}
		return ""
	}
}

func findNeighbors(nodeID uint64, snap *game.StateSnapshot) []string {
	adj := neighborSet(nodeID, snap)
	nids := make([]uint64, 0, len(adj))
	for id := range adj {
		nids = append(nids, id)
	}
	slices.Sort(nids)

	var names []string
	for _, nid := range nids {
		if n, ok := snap.Nodes[nid]; ok {
			names = append(names, n.Label)
		}
	}
	return names
}
