package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dshlychkou/cyberspace/internal/game"
	"github.com/dshlychkou/cyberspace/internal/network"
)

func renderNodeList(snap game.StateSnapshot, selectedIdx int, nodeIDs []uint64, width int) string {
	// Pre-build adjacency for showing connections inline
	adj := buildAdjacency(snap)

	var sb strings.Builder

	// Group nodes by type
	type group struct {
		title string
		hint  string
		ids   []uint64
	}
	groups := []group{
		{"SERVERS", "", nil},
		{"RELAYS", "+2 Compute/tick", nil},
		{"VAULTS", "+5 Data/tick", nil},
		{"FIREWALLS", "blocks auto-spread, ICE spawns here", nil},
		{"CORE", "target — hold to win!", nil},
	}

	for _, id := range nodeIDs {
		n := snap.Nodes[id]
		switch n.Type {
		case network.NodeServer:
			groups[0].ids = append(groups[0].ids, id)
		case network.NodeRelay:
			groups[1].ids = append(groups[1].ids, id)
		case network.NodeVault:
			groups[2].ids = append(groups[2].ids, id)
		case network.NodeFirewall:
			groups[3].ids = append(groups[3].ids, id)
		case network.NodeCore:
			groups[4].ids = append(groups[4].ids, id)
		}
	}

	// Track which index we're at overall
	idx := 0
	for _, g := range groups {
		if len(g.ids) == 0 {
			continue
		}

		// Group header
		header := styleTitle.Render(g.title)
		if g.hint != "" {
			header += styleEvent.Render(" (" + g.hint + ")")
		}
		sb.WriteString("  " + header + "\n")

		for _, id := range g.ids {
			n := snap.Nodes[id]
			programs, ices, viruses := countEntities(n, snap)

			// Selection cursor
			cursor := "  "
			if idx == selectedIdx {
				cursor = styleSelected.Render("> ")
			}

			// Node symbol + label
			sym := network.NodeType(n.Type).Symbol()
			var nodeStr string
			switch n.Type {
			case network.NodeCore:
				nodeStr = styleCore.Render(sym + " " + n.Label)
			case network.NodeFirewall:
				nodeStr = styleFirewall.Render(sym + " " + n.Label)
			case network.NodeVault:
				nodeStr = styleData.Render(sym + " " + n.Label)
			case network.NodeRelay:
				nodeStr = styleRelay.Render(sym + " " + n.Label)
			default:
				nodeStr = styleServer.Render(sym + " " + n.Label)
			}

			// Entity tags
			var tags []string
			if programs > 0 {
				tags = append(tags, styleProgram.Render(fmt.Sprintf("%dP", programs)))
			}
			if ices > 0 {
				tags = append(tags, styleICE.Render(fmt.Sprintf("%dI", ices)))
			}
			if viruses > 0 {
				tags = append(tags, styleVirus.Render(fmt.Sprintf("%dV", viruses)))
			}

			tagStr := ""
			if len(tags) > 0 {
				tagStr = " " + strings.Join(tags, " ")
			}

			// Show connections inline
			connStr := ""
			if neighbors, ok := adj[id]; ok && len(neighbors) > 0 {
				var neighborLabels []string
				for _, nid := range neighbors {
					if nn, ok := snap.Nodes[nid]; ok {
						neighborLabels = append(neighborLabels, nn.Label)
					}
				}
				connStr = styleEvent.Render(" → " + strings.Join(neighborLabels, ", "))
			}

			sb.WriteString(cursor + nodeStr + tagStr + connStr + "\n")
			idx++
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

func buildAdjacency(snap game.StateSnapshot) map[uint64][]uint64 {
	adj := make(map[uint64][]uint64)
	seen := make(map[[2]uint64]bool)
	for _, e := range snap.Edges {
		if !seen[[2]uint64{e.From, e.To}] {
			adj[e.From] = append(adj[e.From], e.To)
			seen[[2]uint64{e.From, e.To}] = true
		}
		if !seen[[2]uint64{e.To, e.From}] {
			adj[e.To] = append(adj[e.To], e.From)
			seen[[2]uint64{e.To, e.From}] = true
		}
	}
	// Sort each neighbor list
	for id := range adj {
		nids := adj[id]
		sort.Slice(nids, func(i, j int) bool { return nids[i] < nids[j] })
	}
	return adj
}

func renderSelectedDetails(snap game.StateSnapshot, selectedIdx int, nodeIDs []uint64) string {
	if selectedIdx >= len(nodeIDs) {
		return ""
	}

	selectedID := nodeIDs[selectedIdx]
	n := snap.Nodes[selectedID]
	programs, ices, viruses := countEntities(n, snap)

	var sb strings.Builder

	sym := network.NodeType(n.Type).Symbol()
	sb.WriteString(styleTitle.Render("SELECTED: " + sym + " " + n.Label))
	sb.WriteByte('\n')

	// Find neighbors
	neighbors := findNeighbors(selectedID, snap)
	if len(neighbors) > 0 {
		sb.WriteString("  Links: " + styleEvent.Render(strings.Join(neighbors, ", ")) + "\n")
	}

	// Show entities
	var entities []string
	if programs > 0 {
		entities = append(entities, styleProgram.Render(fmt.Sprintf("%d Program(s)", programs)))
	}
	if ices > 0 {
		entities = append(entities, styleICE.Render(fmt.Sprintf("%d ICE", ices)))
	}
	if viruses > 0 {
		entities = append(entities, styleVirus.Render(fmt.Sprintf("%d Virus(es)", viruses)))
	}
	if len(entities) == 0 {
		entities = append(entities, styleEvent.Render("empty"))
	}
	sb.WriteString("  Here:  " + strings.Join(entities, ", ") + "\n")

	// Contextual hint based on node type and state
	hint := ""
	switch n.Type {
	case network.NodeFirewall:
		if ices > 0 && programs == 0 {
			hint = fmt.Sprintf(
				"%d ICE here. Deploy a virus on a neighbor to convert ICE into a program, or press S to spawn a program (it survives only if you have more programs than ICE on this node).",
				ices)
		} else if ices > 0 && ices >= programs {
			hint = fmt.Sprintf(
				"%d ICE vs %d programs — ICE outnumbers you! Your programs will be killed. Spawn more programs (S) or deploy a virus (V) on a neighbor to convert ICE.",
				ices, programs)
		} else if programs > 0 {
			hint = "You hold this firewall. Programs cannot auto-spread from here to CORE — select CORE and press S to place one there."
		} else {
			hint = "Programs cannot auto-spread here. Press S to manually place a program (costs Data)."
		}
	case network.NodeCore:
		if programs >= snap.CoreWinThreshold {
			hint = fmt.Sprintf(
				"You have %d programs on CORE! Hold %d+ programs here for %d ticks to win. Keep spawning to defend against ICE.",
				programs, snap.CoreWinThreshold, snap.CoreWinDuration)
		} else if programs > 0 {
			hint = fmt.Sprintf(
				"You have %d program(s) on CORE but need %d. Select this node and press S to spawn more.",
				programs, snap.CoreWinThreshold)
		} else {
			hint = "This is your target! Programs cannot auto-spread here. Select this node and press S to place a program."
		}
	case network.NodeVault:
		if programs > 0 {
			hint = fmt.Sprintf("Earning +%d Data per tick from %d program(s) here.", programs*5, programs)
		} else {
			hint = "Programs on a vault earn Data each tick. Data is used to spawn programs (S)."
		}
	case network.NodeRelay:
		if programs > 0 {
			hint = fmt.Sprintf("Earning +%d Compute per tick from %d program(s) here.", programs*2, programs)
		} else {
			hint = "Programs on a relay earn Compute each tick. Compute is used to deploy viruses (V)."
		}
	default:
		if programs > 0 {
			hint = "Programs here auto-spread to connected servers, relays, and vaults (not firewalls or core)."
		} else {
			hint = "Programs auto-spread here from neighbors. No action needed."
		}
	}
	if hint != "" {
		sb.WriteString("  " + styleEvent.Render(">> ") + hint + "\n")
	}

	return sb.String()
}

func findNeighbors(nodeID uint64, snap game.StateSnapshot) []string {
	// Build adjacency from edges
	adj := make(map[uint64]bool)
	for _, e := range snap.Edges {
		if e.From == nodeID {
			adj[e.To] = true
		}
		if e.To == nodeID {
			adj[e.From] = true
		}
	}

	// Sort neighbor IDs
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
