package tui

import (
	"strings"
	"testing"

	"github.com/dshlychkou/cyberspace/v2/internal/game"
	"github.com/dshlychkou/cyberspace/v2/internal/network"
)

func TestWorkstationBootOnce(t *testing.T) {
	var w workstation
	snap := &game.StateSnapshot{Tick: 0}
	w.sync(snap)
	w.sync(snap)
	boots := 0
	for _, l := range w.lines {
		if l.kind == ckBoot {
			boots++
		}
	}
	if boots != 3 {
		t.Fatalf("expected 3 boot lines once, got %d (total %d)", boots, len(w.lines))
	}
}

func TestWorkstationIngestsEvents(t *testing.T) {
	var w workstation
	snap := &game.StateSnapshot{
		Tick: 1,
		Events: []game.Event{
			{Tick: 1, Message: "program spawned"},
			{Tick: 1, Message: "ICE patrol"},
		},
	}
	w.sync(snap)
	ops := 0
	for _, l := range w.lines {
		if l.kind == ckOps {
			ops++
		}
	}
	if ops != 2 {
		t.Fatalf("want 2 ops lines, got %d", ops)
	}
	// No duplicate on resync
	w.sync(snap)
	ops = 0
	for _, l := range w.lines {
		if l.kind == ckOps {
			ops++
		}
	}
	if ops != 2 {
		t.Fatalf("ops duplicated: %d", ops)
	}
}

func TestWorkstationIdleBreath(t *testing.T) {
	var w workstation
	w.sync(&game.StateSnapshot{Tick: 0})
	w.sync(&game.StateSnapshot{Tick: 8})
	idle := 0
	for _, l := range w.lines {
		if l.kind == ckIdle {
			idle++
		}
	}
	if idle != 1 {
		t.Fatalf("want 1 idle line at tick 8, got %d", idle)
	}
}

func TestRenderWorkstationContainsChrome(t *testing.T) {
	var w workstation
	snap := &game.StateSnapshot{
		Tick:             3,
		CoreWinThreshold: 3,
		CoreWinDuration:  5,
		ProgramSpawnCost: 10,
		VirusDeployCost:  15,
		Nodes: map[uint64]game.NodeSnapshot{
			1: {ID: 1, Label: "vault-1", Type: network.NodeVault},
		},
		Resources: game.Resources{Data: 40, Compute: 20},
	}
	w.sync(snap)
	out := renderWorkstation(snap, 1, 24, 20, &w)
	for _, want := range []string{"SYS://NODE-OPS", "LOCK", "PROBE", "PULSE", "OBJ"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestWorkstationReset(t *testing.T) {
	var w workstation
	w.sync(&game.StateSnapshot{Tick: 1, Events: []game.Event{{Tick: 1, Message: "x"}}})
	w.reset()
	if w.booted || len(w.lines) != 0 || len(w.seen) != 0 {
		t.Fatalf("reset incomplete: %+v", w)
	}
}

func TestWorkstationIngestsSlidingWindow(t *testing.T) {
	var w workstation
	w.sync(&game.StateSnapshot{Tick: 1, Events: []game.Event{
		{Tick: 1, Message: "a"}, {Tick: 1, Message: "b"}, {Tick: 1, Message: "c"},
	}})
	w.sync(&game.StateSnapshot{Tick: 2, Events: []game.Event{
		{Tick: 1, Message: "b"}, {Tick: 1, Message: "c"}, {Tick: 2, Message: "d"},
	}})
	var ops []string
	for _, l := range w.lines {
		if l.kind == ckOps {
			ops = append(ops, l.text)
		}
	}
	want := []string{"[1] a", "[1] b", "[1] c", "[2] d"}
	if strings.Join(ops, "|") != strings.Join(want, "|") {
		t.Fatalf("ops = %v, want %v", ops, want)
	}
}
