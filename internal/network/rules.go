package network

import (
	"github.com/dshlychkou/cyberspace/internal/entity"
)

// RuleConfig holds tunable parameters for Conway-style cellular automata rules.
type RuleConfig struct {
	SurviveMin  int
	SurviveMax  int
	SpreadExact int
}

// RuleResult collects all actions produced by a single rule evaluation pass.
type RuleResult struct {
	Spawns []SpawnAction
	Deaths []int // entity IDs to kill
	Moves  []MoveAction
	Flips  []FlipAction
}

type SpawnAction struct {
	Reason      string
	NodeID      uint64
	Kind        entity.Kind
	NeighborCnt int
}

type MoveAction struct {
	ToNodeID uint64
	EntityID int
}

// FlipAction represents a virus converting an ICE entity into a program.
type FlipAction struct {
	NodeID   uint64
	EntityID int
}

type EntitySnapshot struct {
	NodeID uint64
	ID     int
	Kind   entity.Kind
}

// nodeContext holds pre-computed entity counts and neighbor data for a single
// node, avoiding repeated map lookups during rule evaluation.
type nodeContext struct {
	nodeID           uint64
	node             *Node
	neighbors        []uint64
	localPrograms    int
	localICE         int
	localViruses     int
	neighborPrograms int
	entities         []EntitySnapshot
}

// ruleEvaluator holds the shared index built from entity snapshots and
// produces a RuleResult by running each rule category in sequence.
type ruleEvaluator struct {
	net            *Network
	cfg            RuleConfig
	nodePrograms   map[uint64]int
	nodeICE        map[uint64]int
	nodeViruses    map[uint64]int
	nodeEntitySnap map[uint64][]EntitySnapshot
	result         RuleResult
}

// EvaluateRules runs Conway-style rules across all nodes and returns the
// combined actions (deaths, spawns, moves, flips).
func EvaluateRules(net *Network, entities []EntitySnapshot, cfg RuleConfig) RuleResult {
	ev := &ruleEvaluator{
		net:            net,
		cfg:            cfg,
		nodePrograms:   make(map[uint64]int),
		nodeICE:        make(map[uint64]int),
		nodeViruses:    make(map[uint64]int),
		nodeEntitySnap: make(map[uint64][]EntitySnapshot),
	}
	ev.buildIndex(entities)

	for _, nodeID := range net.NodeIDs() {
		nc := ev.contextFor(nodeID)
		ev.evalProgramSurvival(nc)
		ev.evalAutoSpread(nc)
		ev.evalICESuppress(nc)
		ev.evalICEPatrol(nc)
		ev.evalVirusCorrupt(nc)
	}
	return ev.result
}

// buildIndex populates per-node entity counts and snapshot lists.
func (ev *ruleEvaluator) buildIndex(entities []EntitySnapshot) {
	for _, e := range entities {
		switch e.Kind {
		case entity.KindProgram:
			ev.nodePrograms[e.NodeID]++
		case entity.KindICE:
			ev.nodeICE[e.NodeID]++
		case entity.KindVirus:
			ev.nodeViruses[e.NodeID]++
		}
		ev.nodeEntitySnap[e.NodeID] = append(ev.nodeEntitySnap[e.NodeID], e)
	}
}

// contextFor computes the local and neighbor counts for a single node.
func (ev *ruleEvaluator) contextFor(nodeID uint64) nodeContext {
	neighbors := ev.net.Neighbors(nodeID)
	neighborPrograms := 0
	for _, nid := range neighbors {
		neighborPrograms += ev.nodePrograms[nid]
	}
	return nodeContext{
		nodeID:           nodeID,
		node:             ev.net.GetNode(nodeID),
		neighbors:        neighbors,
		localPrograms:    ev.nodePrograms[nodeID],
		localICE:         ev.nodeICE[nodeID],
		localViruses:     ev.nodeViruses[nodeID],
		neighborPrograms: neighborPrograms,
		entities:         ev.nodeEntitySnap[nodeID],
	}
}

// evalProgramSurvival kills programs that lack neighbor support or are
// outnumbered by ICE on the same node.
func (ev *ruleEvaluator) evalProgramSurvival(nc nodeContext) {
	for _, e := range nc.entities {
		if e.Kind != entity.KindProgram {
			continue
		}
		support := nc.localPrograms - 1 + nc.neighborPrograms
		alive := support >= ev.cfg.SurviveMin && support <= ev.cfg.SurviveMax
		if nc.localICE >= nc.localPrograms {
			alive = false
		}
		if !alive {
			ev.result.Deaths = append(ev.result.Deaths, e.ID)
		}
	}
}

// evalAutoSpread spawns a program on empty, non-fortified nodes (server,
// relay, vault) when enough neighbor programs exist. Firewalls and core
// require manual placement.
func (ev *ruleEvaluator) evalAutoSpread(nc nodeContext) {
	if nc.localPrograms != 0 || nc.localICE != 0 {
		return
	}
	canSpread := nc.node.Type == NodeServer || nc.node.Type == NodeRelay || nc.node.Type == NodeVault
	if canSpread && nc.neighborPrograms >= ev.cfg.SpreadExact {
		ev.result.Spawns = append(ev.result.Spawns, SpawnAction{
			Kind:        entity.KindProgram,
			NodeID:      nc.nodeID,
			Reason:      "auto-spread",
			NeighborCnt: nc.neighborPrograms,
		})
	}
}

// evalICESuppress kills all programs on a node where ICE meets or
// outnumbers them.
func (ev *ruleEvaluator) evalICESuppress(nc nodeContext) {
	if nc.localICE == 0 || nc.localPrograms == 0 || nc.localICE < nc.localPrograms {
		return
	}
	for _, e := range nc.entities {
		if e.Kind == entity.KindProgram {
			ev.result.Deaths = append(ev.result.Deaths, e.ID)
		}
	}
}

// evalICEPatrol moves one ICE toward the first undefended neighbor that
// contains programs.
func (ev *ruleEvaluator) evalICEPatrol(nc nodeContext) {
	if nc.localICE == 0 || nc.neighborPrograms == 0 {
		return
	}
	for _, nid := range nc.neighbors {
		if ev.nodePrograms[nid] > 0 && ev.nodeICE[nid] == 0 {
			for _, e := range nc.entities {
				if e.Kind == entity.KindICE {
					ev.result.Moves = append(ev.result.Moves, MoveAction{
						EntityID: e.ID,
						ToNodeID: nid,
					})
					return
				}
			}
		}
	}
}

// evalVirusCorrupt flips one ICE per adjacent node that contains viruses,
// converting it into a player program.
func (ev *ruleEvaluator) evalVirusCorrupt(nc nodeContext) {
	if nc.localViruses == 0 {
		return
	}
	for _, nid := range nc.neighbors {
		if ev.nodeICE[nid] == 0 {
			continue
		}
		for _, e := range ev.nodeEntitySnap[nid] {
			if e.Kind == entity.KindICE {
				ev.result.Flips = append(ev.result.Flips, FlipAction{
					EntityID: e.ID,
					NodeID:   nid,
				})
				break
			}
		}
	}
}
