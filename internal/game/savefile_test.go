package game

import (
	"encoding/json"
	"testing"
)

func mustFromSaveFile(t *testing.T, sf *SaveFile) *State {
	t.Helper()
	state, err := FromSaveFile(sf)
	if err != nil {
		t.Fatalf("FromSaveFile: %v", err)
	}
	return state
}

func TestSaveFileRoundtrip(t *testing.T) {
	cfg := testConfig()
	cfg.EventLogFile = "" // no log file for tests
	state := mustInitGame(t, &cfg)

	// Run a few ticks
	for range 5 {
		state.Tick++
		state.processScheduledEvents()
		state.ageViruses()
		state.applyRules()
		state.processEconomy()
	}

	// Save
	sf := state.ToSaveFile()

	// Marshal/unmarshal
	data, err := json.Marshal(sf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var sf2 SaveFile
	if err := json.Unmarshal(data, &sf2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Restore
	restored := mustFromSaveFile(t, &sf2)

	// Verify fields match
	if restored.Tick != state.Tick {
		t.Errorf("tick: want %d, got %d", state.Tick, restored.Tick)
	}
	if restored.Score != state.Score {
		t.Errorf("score: want %d, got %d", state.Score, restored.Score)
	}
	if restored.CoreHoldLen != state.CoreHoldLen {
		t.Errorf("coreHoldLen: want %d, got %d", state.CoreHoldLen, restored.CoreHoldLen)
	}
	if restored.Resources != state.Resources {
		t.Errorf("resources: want %+v, got %+v", state.Resources, restored.Resources)
	}
	if restored.Paused != state.Paused {
		t.Errorf("paused: want %v, got %v", state.Paused, restored.Paused)
	}
	if restored.GameOver != state.GameOver {
		t.Errorf("gameOver: want %v, got %v", state.GameOver, restored.GameOver)
	}
	if restored.Won != state.Won {
		t.Errorf("won: want %v, got %v", state.Won, restored.Won)
	}
	if len(restored.Programs) != len(state.Programs) {
		t.Errorf("programs: want %d, got %d", len(state.Programs), len(restored.Programs))
	}
	if len(restored.ICEs) != len(state.ICEs) {
		t.Errorf("ICEs: want %d, got %d", len(state.ICEs), len(restored.ICEs))
	}
	if len(restored.Viruses) != len(state.Viruses) {
		t.Errorf("viruses: want %d, got %d", len(state.Viruses), len(restored.Viruses))
	}
	if len(restored.Network.Nodes) != len(state.Network.Nodes) {
		t.Errorf("nodes: want %d, got %d", len(state.Network.Nodes), len(restored.Network.Nodes))
	}
	if restored.nextEntityID != state.nextEntityID {
		t.Errorf("nextEntityID: want %d, got %d", state.nextEntityID, restored.nextEntityID)
	}

	// Verify entity IDs exist in maps
	for id := range state.Programs {
		if _, ok := restored.Programs[id]; !ok {
			t.Errorf("program %d missing in restored state", id)
		}
	}
	for id := range state.ICEs {
		if _, ok := restored.ICEs[id]; !ok {
			t.Errorf("ICE %d missing in restored state", id)
		}
	}

	// Verify scheduler was restored
	if restored.sched == nil {
		t.Fatal("scheduler should not be nil")
	}
	if restored.sched.Size() != state.sched.Size() {
		t.Errorf("scheduler size: want %d, got %d", state.sched.Size(), restored.sched.Size())
	}

	// Verify restored state can advance ticks
	restored.Tick++
	restored.applyRules()
	restored.processEconomy()
	// Should not panic
}

func TestSaveFileNodeEntities(t *testing.T) {
	cfg := testConfig()
	cfg.EventLogFile = ""
	state := mustInitGame(t, &cfg)

	// Pick a node with entities
	var nodeWithEntities uint64
	for id, n := range state.Network.Nodes {
		if len(n.Entities) > 0 {
			nodeWithEntities = id
			break
		}
	}
	if nodeWithEntities == 0 {
		t.Skip("no node with entities found")
	}

	sf := state.ToSaveFile()
	data, err := json.Marshal(sf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var sf2 SaveFile
	if err := json.Unmarshal(data, &sf2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	restored := mustFromSaveFile(t, &sf2)

	origNode := state.Network.GetNode(nodeWithEntities)
	restoredNode := restored.Network.GetNode(nodeWithEntities)
	if restoredNode == nil {
		t.Fatalf("node %d not found in restored state", nodeWithEntities)
	}
	if len(restoredNode.Entities) != len(origNode.Entities) {
		t.Errorf("node %d entities: want %d, got %d",
			nodeWithEntities, len(origNode.Entities), len(restoredNode.Entities))
	}
}
