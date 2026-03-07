package network

import (
	"testing"

	"github.com/dshlychkou/cyberspace/internal/entity"
)

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

	cfg := RuleConfig{SurviveMin: 1, SurviveMax: 6, SpreadExact: 2}
	result := EvaluateRules(net, entities, cfg)

	// Program on node 1 has 2 program neighbors (nodes 2,3) -> survives
	for _, id := range result.Deaths {
		if id == 1 {
			t.Error("program on node 1 should survive with 2 program neighbors")
		}
	}
}

func TestProgramSpread(t *testing.T) {
	net := New()
	for i := 1; i <= 4; i++ {
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

	cfg := RuleConfig{SurviveMin: 1, SurviveMax: 6, SpreadExact: 2}
	result := EvaluateRules(net, entities, cfg)

	// Node 4 has 3 program neighbors (>= SpreadExact=2) -> should spawn
	found := false
	for _, spawn := range result.Spawns {
		if spawn.NodeID == 4 && spawn.Kind == entity.KindProgram {
			found = true
		}
	}
	if !found {
		t.Error("expected program to spread to node 4 (3 program neighbors >= SpreadExact 2)")
	}
}

func TestICESuppression(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindICE, NodeID: 1},
		{ID: 2, Kind: entity.KindICE, NodeID: 1},
		{ID: 3, Kind: entity.KindProgram, NodeID: 1},
	}

	cfg := RuleConfig{SurviveMin: 1, SurviveMax: 6, SpreadExact: 2}
	result := EvaluateRules(net, entities, cfg)

	// ICE (2) > Programs (1) on same node -> program should die
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

func TestFirewallBlocksAutoSpread(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.AddNode(NewNode(3, NodeFirewall)) // firewall should block auto-spread
	net.Connect(3, 1)
	net.Connect(3, 2)

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
	}

	cfg := RuleConfig{SurviveMin: 1, SurviveMax: 6, SpreadExact: 2}
	result := EvaluateRules(net, entities, cfg)

	// Node 3 (firewall) has 2 program neighbors but should NOT get auto-spread
	for _, spawn := range result.Spawns {
		if spawn.NodeID == 3 {
			t.Error("firewall should block auto-spread of programs")
		}
	}
}

func TestCoreBlocksAutoSpread(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.AddNode(NewNode(3, NodeCore)) // core should block auto-spread
	net.Connect(3, 1)
	net.Connect(3, 2)

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindProgram, NodeID: 1},
		{ID: 2, Kind: entity.KindProgram, NodeID: 2},
	}

	cfg := RuleConfig{SurviveMin: 1, SurviveMax: 6, SpreadExact: 2}
	result := EvaluateRules(net, entities, cfg)

	// Node 3 (core) has 2 program neighbors but should NOT get auto-spread
	for _, spawn := range result.Spawns {
		if spawn.NodeID == 3 {
			t.Error("core should block auto-spread of programs")
		}
	}
}

func TestVirusCorruption(t *testing.T) {
	net := New()
	net.AddNode(NewNode(1, NodeServer))
	net.AddNode(NewNode(2, NodeServer))
	net.Connect(1, 2)

	entities := []EntitySnapshot{
		{ID: 1, Kind: entity.KindVirus, NodeID: 1},
		{ID: 2, Kind: entity.KindICE, NodeID: 2},
	}

	cfg := RuleConfig{SurviveMin: 1, SurviveMax: 6, SpreadExact: 2}
	result := EvaluateRules(net, entities, cfg)

	// Virus on node 1 should corrupt adjacent ICE on node 2
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
