package tui

import (
	"math"
	"testing"

	"github.com/dshlychkou/cyberspace/internal/game"
	"github.com/dshlychkou/cyberspace/internal/network"
)

func TestLayoutPlacesDistinctNodes(t *testing.T) {
	snap := &game.StateSnapshot{
		Nodes: map[uint64]game.NodeSnapshot{
			1: {ID: 1, Type: network.NodeCore, Label: "CORE"},
			2: {ID: 2, Type: network.NodeFirewall, Label: "FW2"},
			3: {ID: 3, Type: network.NodeServer, Label: "S3"},
			4: {ID: 4, Type: network.NodeVault, Label: "V4"},
			5: {ID: 5, Type: network.NodeRelay, Label: "R5"},
		},
	}
	var cam camera
	cam.reset()
	pos := layoutNodes(snap, 80, 24, cam)
	if len(pos) != 5 {
		t.Fatalf("want 5 positions, got %d", len(pos))
	}
	seen := map[[2]int]bool{}
	for _, p := range pos {
		key := [2]int{p.x, p.y}
		if seen[key] {
			t.Fatalf("two nodes share cell %v — topology unreadable", key)
		}
		seen[key] = true
	}
}

func TestYawRotatesNonCoreNodes(t *testing.T) {
	snap := &game.StateSnapshot{
		Nodes: map[uint64]game.NodeSnapshot{
			1: {ID: 1, Type: network.NodeCore, Label: "CORE"},
			2: {ID: 2, Type: network.NodeServer, Label: "S2"},
			3: {ID: 3, Type: network.NodeVault, Label: "V3"},
		},
	}
	var cam camera
	cam.reset()
	a := indexPos(layoutNodes(snap, 80, 24, cam))
	cam.orbitYaw(1)
	b := indexPos(layoutNodes(snap, 80, 24, cam))

	if a[1].x != b[1].x || a[1].y != b[1].y {
		t.Fatal("CORE should stay centered when rotating")
	}
	d := math.Hypot(float64(a[2].x-b[2].x), float64(a[2].y-b[2].y))
	if d < 2 {
		t.Fatalf("server should move on yaw, delta=%.1f", d)
	}
}

func TestNeighborSet(t *testing.T) {
	snap := &game.StateSnapshot{
		Edges: []game.EdgeSnapshot{
			{From: 1, To: 2},
			{From: 2, To: 3},
		},
	}
	n := neighborSet(2, snap)
	if !n[1] || !n[3] || n[2] {
		t.Fatalf("unexpected neighbors: %v", n)
	}
}

func indexPos(pos []nodePos) map[uint64]nodePos {
	out := make(map[uint64]nodePos, len(pos))
	for _, p := range pos {
		out[p.id] = p
	}
	return out
}
