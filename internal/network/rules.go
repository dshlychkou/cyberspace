package network

import (
	"github.com/dshlychkou/cyberspace/internal/entity"
)

type RuleResult struct {
	Spawns []SpawnAction
	Deaths []int // entity IDs to kill
	Moves  []MoveAction
	Flips  []FlipAction // virus converts ICE to program
}

type SpawnAction struct {
	Kind        entity.Kind
	NodeID      uint64
	Reason      string
	NeighborCnt int
}

type MoveAction struct {
	EntityID int
	ToNodeID uint64
}

type FlipAction struct {
	EntityID int // ICE entity to convert
	NodeID   uint64
}

type EntitySnapshot struct {
	ID     int
	Kind   entity.Kind
	NodeID uint64
}

type DeathReason int

const (
	DeathIsolation DeathReason = iota
	DeathOvercrowding
	DeathICESuppress
)

type DeathAction struct {
	EntityID int
	Reason   DeathReason
}

func EvaluateRules(net *Network, entities []EntitySnapshot, cfg RuleConfig) RuleResult {
	var result RuleResult

	// Build per-node entity counts
	nodePrograms := make(map[uint64]int)
	nodeICE := make(map[uint64]int)
	nodeViruses := make(map[uint64]int)
	nodeEntityIDs := make(map[uint64][]EntitySnapshot)

	for _, e := range entities {
		switch e.Kind {
		case entity.KindProgram:
			nodePrograms[e.NodeID]++
		case entity.KindICE:
			nodeICE[e.NodeID]++
		case entity.KindVirus:
			nodeViruses[e.NodeID]++
		}
		nodeEntityIDs[e.NodeID] = append(nodeEntityIDs[e.NodeID], e)
	}

	// Evaluate each node
	for _, nodeID := range net.NodeIDs() {
		node := net.GetNode(nodeID)
		neighbors := net.Neighbors(nodeID)

		// Count neighbor entities
		neighborPrograms := 0
		neighborICE := 0
		for _, nid := range neighbors {
			neighborPrograms += nodePrograms[nid]
			neighborICE += nodeICE[nid]
		}

		// --- Program rules ---
		localPrograms := nodePrograms[nodeID]
		for _, e := range nodeEntityIDs[nodeID] {
			if e.Kind != entity.KindProgram {
				continue
			}
			alive := true

			// Support = co-located programs (excluding self) + neighbor programs
			support := localPrograms - 1 + neighborPrograms

			// Safe zones: vaults and relays don't need neighbor support
			surviveMin := cfg.SurviveMin
			if node.Type == NodeVault || node.Type == NodeRelay {
				surviveMin = 0
			}

			if support < surviveMin || support > cfg.SurviveMax {
				alive = false
			}
			// ICE on same node outnumbers local programs → die
			if nodeICE[nodeID] > localPrograms {
				alive = false
			}
			if !alive {
				result.Deaths = append(result.Deaths, e.ID)
			}
		}

		// --- Auto-spread rule ---
		// Programs auto-spread ONLY to server/relay/vault nodes (NOT firewalls/core)
		// Firewalls and core require manual placement
		if nodePrograms[nodeID] == 0 && nodeICE[nodeID] == 0 {
			canAutoSpread := node.Type == NodeServer || node.Type == NodeRelay || node.Type == NodeVault
			if canAutoSpread && neighborPrograms >= cfg.SpreadExact {
				result.Spawns = append(result.Spawns, SpawnAction{
					Kind:        entity.KindProgram,
					NodeID:      nodeID,
					Reason:      "auto-spread",
					NeighborCnt: neighborPrograms,
				})
			}
		}

		// --- ICE rules ---
		// Suppress: ICE kills programs where ICE > programs on same node
		if nodeICE[nodeID] > 0 && nodePrograms[nodeID] > 0 {
			if nodeICE[nodeID] > nodePrograms[nodeID] {
				for _, e := range nodeEntityIDs[nodeID] {
					if e.Kind == entity.KindProgram {
						result.Deaths = append(result.Deaths, e.ID)
					}
				}
			}
		}

		// ICE patrol: move toward nodes with programs
		if nodeICE[nodeID] > 0 && neighborPrograms > 0 {
			for _, nid := range neighbors {
				if nodePrograms[nid] > 0 && nodeICE[nid] == 0 {
					for _, e := range nodeEntityIDs[nodeID] {
						if e.Kind == entity.KindICE {
							result.Moves = append(result.Moves, MoveAction{
								EntityID: e.ID,
								ToNodeID: nid,
							})
							break
						}
					}
					break
				}
			}
		}

		// --- Virus rules ---
		// Corrupt: virus converts adjacent ICE to programs
		if nodeViruses[nodeID] > 0 {
			for _, nid := range neighbors {
				if nodeICE[nid] > 0 {
					for _, e := range nodeEntityIDs[nid] {
						if e.Kind == entity.KindICE {
							result.Flips = append(result.Flips, FlipAction{
								EntityID: e.ID,
								NodeID:   nid,
							})
							break
						}
					}
				}
			}
		}
	}

	return result
}

type RuleConfig struct {
	SurviveMin  int
	SurviveMax  int
	SpreadExact int
}
