package game

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/barnowlsnest/go-actorlib/v4/pkg/actor"
	"github.com/barnowlsnest/go-actorlib/v4/pkg/middleware"
)

func TestInitGame(t *testing.T) {
	cfg := DefaultConfig()
	state := InitGame(cfg)

	if state == nil {
		t.Fatal("expected non-nil state")
	}

	if len(state.Network.Nodes) == 0 {
		t.Error("expected generated network to have nodes")
	}

	if len(state.Programs) < 1 {
		t.Errorf("expected at least 1 initial program, got %d", len(state.Programs))
	}

	if len(state.ICEs) < 1 {
		t.Error("expected at least 1 initial ICE")
	}

	if state.Resources.Data != 100 {
		t.Errorf("expected 100 initial data, got %d", state.Resources.Data)
	}
}

func TestSpawnProgram(t *testing.T) {
	cfg := DefaultConfig()
	state := InitGame(cfg)

	// Pick a node to spawn on
	var nodeID uint64
	for id := range state.Network.Nodes {
		nodeID = id
		break
	}

	initialData := state.Resources.Data
	state.AddProgram(nodeID)

	initialCount := len(state.Programs) - 1 // before AddProgram
	_ = initialCount
	if len(state.Programs) < 2 {
		t.Errorf("expected at least 2 programs after spawn, got %d", len(state.Programs))
	}

	// Resources should not be deducted by AddProgram (only by SpawnProgramCmd)
	if state.Resources.Data != initialData {
		t.Error("AddProgram should not deduct resources")
	}
}

func TestStateSnapshot(t *testing.T) {
	cfg := DefaultConfig()
	state := InitGame(cfg)
	snap := state.Snapshot()

	if snap.Tick != 0 {
		t.Errorf("expected tick 0, got %d", snap.Tick)
	}

	if len(snap.Nodes) != len(state.Network.Nodes) {
		t.Errorf("snapshot nodes mismatch: %d vs %d", len(snap.Nodes), len(state.Network.Nodes))
	}

	if len(snap.Programs) != len(state.Programs) {
		t.Errorf("snapshot programs mismatch: %d vs %d", len(snap.Programs), len(state.Programs))
	}
}

type stateProvider struct{ state *State }

func (p *stateProvider) Provide() *State { return p.state }

func startTestActor(t *testing.T, state *State) *actor.GoActor[*State] {
	t.Helper()
	ctx := context.Background()
	a, err := actor.StartNew[*State](
		ctx,
		5*time.Second,
		actor.WithProvider[*State](&stateProvider{state: state}),
		actor.WithName[*State]("test-engine"),
		actor.WithInputBufferSize[*State](16),
		actor.WithMiddleware(middleware.Recovery[*State](slog.Default())),
	)
	if err != nil {
		t.Fatalf("failed to start actor: %v", err)
	}
	t.Cleanup(func() { _ = a.Stop(5 * time.Second) })
	return a
}

func TestTickCmdViaActor(t *testing.T) {
	state := InitGame(DefaultConfig())
	a := startTestActor(t, state)
	ctx := context.Background()

	done := make(chan StateSnapshot, 1)
	cmd := &TickCmd{OnComplete: func(snap StateSnapshot) { done <- snap }}

	if err := a.Receive(ctx, cmd); err != nil {
		t.Fatalf("receive error: %v", err)
	}

	select {
	case snap := <-done:
		if snap.Tick != 1 {
			t.Errorf("expected tick 1, got %d", snap.Tick)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("tick command timed out — possible deadlock")
	}
}

func TestSpawnProgramCmdViaActor(t *testing.T) {
	state := InitGame(DefaultConfig())
	a := startTestActor(t, state)
	ctx := context.Background()

	var nodeID uint64
	for id := range state.Network.Nodes {
		nodeID = id
		break
	}

	done := make(chan string, 1)
	cmd := &SpawnProgramCmd{
		NodeID: nodeID,
		OnComplete: func(ok bool, msg string) {
			if !ok {
				done <- msg
			} else {
				done <- ""
			}
		},
	}

	if err := a.Receive(ctx, cmd); err != nil {
		t.Fatalf("receive error: %v", err)
	}

	select {
	case msg := <-done:
		if msg != "" {
			t.Fatalf("spawn failed: %s", msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("spawn command timed out — possible deadlock")
	}
}

func TestMultipleTicksViaActor(t *testing.T) {
	state := InitGame(DefaultConfig())
	a := startTestActor(t, state)
	ctx := context.Background()

	for i := range 10 {
		done := make(chan StateSnapshot, 1)
		cmd := &TickCmd{OnComplete: func(snap StateSnapshot) { done <- snap }}

		if err := a.Receive(ctx, cmd); err != nil {
			t.Fatalf("tick %d receive error: %v", i, err)
		}

		select {
		case snap := <-done:
			if snap.GameOver {
				t.Logf("game ended at tick %d (won=%v), stopping", snap.Tick, snap.Won)
				return
			}
			if snap.Tick != i+1 {
				t.Errorf("tick %d: expected tick %d, got %d", i, i+1, snap.Tick)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("tick %d timed out — possible deadlock", i)
		}
	}
}
