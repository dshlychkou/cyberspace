package network

import (
	"testing"

	"github.com/dshlychkou/cyberspace/internal/entity"
)

var defaultCfg = RuleConfig{SurviveMin: 1, SurviveMax: 10, SpreadExact: 3}

// --- Program Survival ---

func TestProgramSurvival(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.AddNode(NewNode(3, NodeServer))
	net.AddNode(NewNode(4, NodeServer))
	net.Connect(1, 2)
	net.Connect(1, 3)
	net.Connect(1, 4)

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
		{ID: 3, Kind: entity.KindProgram, NodeID: 3},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, id := range result.Deaths {
		if id == 1 {
			t.Error("program on node 1 should survive with 2 program neighbors")
		}
	}
}

func TestProgramDiesWhenIsolated(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.Connect(1, 2)

	// Single program, no neighbors with programs -> support = 0 < SurviveMin(1)
	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	found := false
	for _, id := range result.Deaths {
		if id == 1 {
			found = true
		}
	}
	if !found {
		t.Error("isolated program (0 support) should die")
	}
}

func TestProgramDiesFromOvercrowding(t *testing.T) {
	// Star topology: center node connected to 11 neighbors, all with programs
	net := New()
	for i := uint64(1); i <= 12; i++ {
		net.AddNode(NewNode(i, NodeServer))
	}
	for i := uint64(2); i <= 12; i++ {
		net.Connect(1, i)
	}

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
	}
	for i := 2; i <= 12; i++ {
		entities = append(entities, EntitySnapshot{ID: i, Kind: entity.KindProgram, NodeID: uint64(i)})
	}

	// Program on node 1: support = 0 (local-1) + 11 (neighbors) = 11 > SurviveMax(10)
	result := EvaluateRules(net, entities, defaultCfg)

	found := false
	for _, id := range result.Deaths {
		if id == 1 {
			found = true
		}
	}
	if !found {
		t.Error("program with 11 neighbor support should die from overcrowding (max 10)")
	}
}

func TestProgramSurvivesAtSurviveMax(t *testing.T) {
	// Center node connected to 10 neighbors, all with programs
	net := New()
	for i := uint64(1); i <= 11; i++ {
		net.AddNode(NewNode(i, NodeServer))
	}
	for i := uint64(2); i <= 11; i++ {
		net.Connect(1, i)
	}

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
	}
	for i := 2; i <= 11; i++ {
		entities = append(entities, EntitySnapshot{ID: i, Kind: entity.KindProgram, NodeID: uint64(i)})
	}

	// Program on node 1: support = 0 + 10 = 10 = SurviveMax -> survives
	result := EvaluateRules(net, entities, defaultCfg)

	for _, id := range result.Deaths {
		if id == 1 {
			t.Error("program with exactly 10 support (= SurviveMax) should survive")
		}
	}
}

func TestColocatedProgramsSurvive(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))

	// 3 programs on same node: each has support = 3-1 = 2
	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 1},
		{ID: 3, Kind: entity.KindProgram, NodeID: 1},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	if len(result.Deaths) > 0 {
		t.Errorf("3 colocated programs (support=2 each) should all survive, got %d deaths", len(result.Deaths))
	}
}

// --- Auto-Spread ---

func TestProgramSpread(t *testing.T) {
	net := New()
	for i := uint64(1); i <= 4; i++ {
		net.AddNode(NewNode(i, NodeServer))
	}
	net.Connect(4, 1)
	net.Connect(4, 2)
	net.Connect(4, 3)

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
		{ID: 3, Kind: entity.KindProgram, NodeID: 3},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	found := false
	for _, spawn := range result.Spawns {
		if spawn.NodeID == 4 && spawn.Kind == entity.KindProgram {
			found = true
		}
	}
	if !found {
		t.Error("expected program to spread to node 4 (3 program neighbors >= SpreadExact 3)")
	}
}

func TestAutoSpreadBlockedByOvercrowding(t *testing.T) {
	// Node 12 connected to 11 nodes with programs -> neighborPrograms=11 > SurviveMax=10
	net := New()
	for i := uint64(1); i <= 12; i++ {
		net.AddNode(NewNode(i, NodeServer))
	}
	for i := uint64(1); i <= 11; i++ {
		net.Connect(12, i)
	}

	entities := make([]EntitySnapshot, 11)
	for i := range 11 {
		entities[i] = EntitySnapshot{ID: i + 1, Kind: entity.KindProgram, NodeID: uint64(i + 1)}
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, spawn := range result.Spawns {
		if spawn.NodeID == 12 {
			t.Error("should NOT auto-spread to node with 11 neighbor programs (exceeds SurviveMax 10)")
		}
	}
}

func TestAutoSpreadAllowedAtSurviveMax(t *testing.T) {
	// Node 11 connected to 10 nodes with programs -> neighborPrograms=10 = SurviveMax -> allowed
	net := New()
	for i := uint64(1); i <= 11; i++ {
		net.AddNode(NewNode(i, NodeServer))
	}
	for i := uint64(1); i <= 10; i++ {
		net.Connect(11, i)
	}

	entities := make([]EntitySnapshot, 10)
	for i := range 10 {
		entities[i] = EntitySnapshot{ID: i + 1, Kind: entity.KindProgram, NodeID: uint64(i + 1)}
	}

	result := EvaluateRules(net, entities, defaultCfg)

	found := false
	for _, spawn := range result.Spawns {
		if spawn.NodeID == 11 {
			found = true
		}
	}
	if !found {
		t.Error("should auto-spread to node with 10 neighbor programs (= SurviveMax)")
	}
}

func TestAutoSpreadBelowThreshold(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.AddNode(NewNode(3, NodeServer))
	net.Connect(3, 1)
	net.Connect(3, 2)

	// Only 2 neighbor programs < SpreadExact(3) -> no spread
	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, spawn := range result.Spawns {
		if spawn.NodeID == 3 {
			t.Error("should NOT spread with only 2 neighbors (SpreadExact=3)")
		}
	}
}

func TestAutoSpreadBlockedByExistingEntities(t *testing.T) {
	net := New()
	for i := uint64(1); i <= 4; i++ {
		net.AddNode(NewNode(i, NodeServer))
	}
	net.Connect(4, 1)
	net.Connect(4, 2)
	net.Connect(4, 3)

	// Node 4 already has an ICE -> no spread
	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
		{ID: 3, Kind: entity.KindProgram, NodeID: 3},
		{ID: 4, Kind: entity.KindICE, NodeID: 4},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, spawn := range result.Spawns {
		if spawn.NodeID == 4 {
			t.Error("should NOT spread to node with existing ICE")
		}
	}
}

// --- Node type restrictions ---

func TestFirewallBlocksAutoSpread(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.AddNode(NewNode(3, NodeServer))
	net.AddNode(NewNode(4, NodeFirewall))
	net.Connect(4, 1)
	net.Connect(4, 2)
	net.Connect(4, 3)

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
		{ID: 3, Kind: entity.KindProgram, NodeID: 3},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, spawn := range result.Spawns {
		if spawn.NodeID == 4 {
			t.Error("firewall should block auto-spread of programs")
		}
	}
}

func TestCoreBlocksAutoSpread(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.AddNode(NewNode(3, NodeServer))
	net.AddNode(NewNode(4, NodeCore))
	net.Connect(4, 1)
	net.Connect(4, 2)
	net.Connect(4, 3)

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
		{ID: 3, Kind: entity.KindProgram, NodeID: 3},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, spawn := range result.Spawns {
		if spawn.NodeID == 4 {
			t.Error("core should block auto-spread of programs")
		}
	}
}

// --- ICE ---

func TestICESuppression(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindICE, NodeID: 1},
		{ID: 2, Kind: entity.KindICE, NodeID: 1},
		{ID: 3, Kind: entity.KindProgram, NodeID: 1},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	found := false
	for _, id := range result.Deaths {
		if id == 3 {
			found = true
		}
	}
	if !found {
		t.Error("expected ICE to suppress program on node 1")
	}
}

func TestICEEqualDoesNotSuppress(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.Connect(1, 2)

	// 1 ICE vs 1 program: programs hold when evenly matched
	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindICE, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 1},
		{ID: 3, Kind: entity.KindProgram, NodeID: 2}, // neighbor support so it doesn't die from isolation
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, id := range result.Deaths {
		if id == 2 {
			t.Error("program should survive when ICE count equals program count (1I vs 1P)")
		}
	}
}

func TestProgramsOutnumberICE(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.Connect(1, 2)

	// 1 ICE vs 2 programs on same node -> programs survive
	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindICE, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 1},
		{ID: 3, Kind: entity.KindProgram, NodeID: 1},
		{ID: 4, Kind: entity.KindProgram, NodeID: 2}, // neighbor support
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, id := range result.Deaths {
		if id == 2 || id == 3 {
			t.Error("programs (2) outnumber ICE (1) — should not be suppressed")
		}
	}
}

func TestICEPatrolsTowardPrograms(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeFirewall))
	net.AddNode(NewNode(2, NodeServer))
	net.Connect(1, 2)

	// ICE on node 1, programs on node 2 (undefended) -> ICE should patrol
	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindICE, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
		{ID: 3, Kind: entity.KindProgram, NodeID: 2},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	found := false
	for _, move := range result.Moves {
		if move.EntityID == 1 && move.ToNodeID == 2 {
			found = true
		}
	}
	if !found {
		t.Error("ICE should patrol toward neighbor with undefended programs")
	}
}

func TestICEDoesNotPatrolToDefendedNode(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeFirewall))
	net.AddNode(NewNode(2, NodeServer))
	net.Connect(1, 2)

	// ICE on both nodes -> no patrol (node 2 already defended)
	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindICE, NodeID: 1},
		{ID: 2, Kind: entity.KindICE, NodeID: 2},
		{ID: 3, Kind: entity.KindProgram, NodeID: 2},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, move := range result.Moves {
		if move.EntityID == 1 {
			t.Error("ICE should NOT patrol to a node that already has ICE")
		}
	}
}

// --- Virus ---

func TestVirusCorruption(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.Connect(1, 2)

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindVirus, NodeID: 1},
		{ID: 2, Kind: entity.KindICE, NodeID: 2},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	found := false
	for _, flip := range result.Flips {
		if flip.EntityID == 2 {
			found = true
		}
	}
	if !found {
		t.Error("expected virus to corrupt adjacent ICE")
	}
}

func TestVirusDoesNotCorruptNonAdjacentICE(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.AddNode(NewNode(3, NodeServer))
	net.Connect(1, 2)
	net.Connect(2, 3)
	// 1 and 3 are NOT connected

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindVirus, NodeID: 1},
		{ID: 2, Kind: entity.KindICE, NodeID: 3},
	}

	result := EvaluateRules(net, entities, defaultCfg)

	for _, flip := range result.Flips {
		if flip.EntityID == 2 {
			t.Error("virus should NOT corrupt ICE on non-adjacent node")
		}
	}
}
