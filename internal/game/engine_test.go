package game

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/barnowlsnest/go-actorlib/v4/pkg/actor"
	"github.com/barnowlsnest/go-actorlib/v4/pkg/middleware"

	"github.com/dshlychkou/cyberspace/internal/network"
)

func testConfig() Config {
	return Config{
		TickRate:            1 * time.Second,
		InitialPrograms:     5,
		InitialICE:          3,
		VirusLifespan:       8,
		CoreWinThreshold:    3,
		CoreWinDuration:     8,
		DataHarvestRate:     5,
		ComputeHarvestRate:  3,
		ProgramSpawnCost:    12,
		VirusDeployCost:     15,
		ProgramUpkeep:       1,
		CoreHoldCost:        2,
		SurviveMin:          1,
		SurviveMax:          10,
		SpreadExact:         3,
		InitialData:         150,
		InitialCompute:      60,
		ICESpawnTick:        999, // disable automatic ICE for deterministic tests
		ICESpawnMinInterval: 8,
		ICEEscalationTick:   999,
		ICEEscalationRate:   50,
	}
}

// --- Init ---

func TestInitGame(t *testing.T) {
	cfg := testConfig()
	state := InitGame(&cfg)

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
	if state.Resources.Data != cfg.InitialData {
		t.Errorf("expected %d initial data, got %d", cfg.InitialData, state.Resources.Data)
	}
	if state.Resources.Compute != cfg.InitialCompute {
		t.Errorf("expected %d initial compute, got %d", cfg.InitialCompute, state.Resources.Compute)
	}
}

func TestSpawnProgram(t *testing.T) {
	cfg := testConfig()
	state := InitGame(&cfg)

	var nodeID uint64
	for id := range state.Network.Nodes {
		nodeID = id
		break
	}

	initialData := state.Resources.Data
	state.AddProgram(nodeID)

	if len(state.Programs) < 2 {
		t.Errorf("expected at least 2 programs after spawn, got %d", len(state.Programs))
	}
	if state.Resources.Data != initialData {
		t.Error("AddProgram should not deduct resources")
	}
}

func TestStateSnapshot(t *testing.T) {
	cfg := testConfig()
	state := InitGame(&cfg)
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
	if snap.CoreWinThreshold != cfg.CoreWinThreshold {
		t.Errorf("snapshot CoreWinThreshold mismatch: %d vs %d", snap.CoreWinThreshold, cfg.CoreWinThreshold)
	}
}

// --- Actor integration ---

type stateProvider struct{ state *State }

func (p *stateProvider) Provide() *State { return p.state }

func startTestActor(t *testing.T, state *State) *actor.GoActor[*State] {
	t.Helper()
	ctx := context.Background()
	a, err := actor.StartNew(
		ctx,
		5*time.Second,
		actor.WithProvider(&stateProvider{state: state}),
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
	cfg := testConfig()
	state := InitGame(&cfg)
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
	cfg := testConfig()
	state := InitGame(&cfg)
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
	cfg := testConfig()
	state := InitGame(&cfg)
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

// --- Economy ---

func newTestState(t *testing.T) *State {
	t.Helper()
	cfg := testConfig()
	net := network.New()
	net.AddNode(NewTestNode(1, network.NodeServer))
	net.AddNode(NewTestNode(2, network.NodeVault))
	net.AddNode(NewTestNode(3, network.NodeRelay))
	net.AddNode(NewTestNode(4, network.NodeCore))
	net.AddNode(NewTestNode(5, network.NodeFirewall))
	net.Connect(1, 2)
	net.Connect(1, 3)
	net.Connect(1, 4)
	net.Connect(4, 5)
	net.Connect(2, 3)

	s := NewState(net, &cfg)
	s.rng = nil // not needed for direct tests
	return s
}

func NewTestNode(id uint64, nodeType network.NodeType) *network.Node {
	return network.NewNode(id, nodeType)
}

func TestHarvestVault(t *testing.T) {
	s := newTestState(t)
	s.AddProgram(2) // vault
	initialData := s.Resources.Data

	s.harvestResources()

	expected := initialData + s.Config.DataHarvestRate
	if s.Resources.Data != expected {
		t.Errorf("vault harvest: expected data=%d, got %d", expected, s.Resources.Data)
	}
}

func TestHarvestRelay(t *testing.T) {
	s := newTestState(t)
	s.AddProgram(3) // relay
	initialCompute := s.Resources.Compute

	s.harvestResources()

	expected := initialCompute + s.Config.ComputeHarvestRate
	if s.Resources.Compute != expected {
		t.Errorf("relay harvest: expected compute=%d, got %d", expected, s.Resources.Compute)
	}
}

func TestHarvestServerGivesNothing(t *testing.T) {
	s := newTestState(t)
	s.AddProgram(1) // server
	beforeData := s.Resources.Data
	beforeCompute := s.Resources.Compute

	s.harvestResources()

	if s.Resources.Data != beforeData {
		t.Error("server should not produce data")
	}
	if s.Resources.Compute != beforeCompute {
		t.Error("server should not produce compute")
	}
}

func TestUpkeepDeductsData(t *testing.T) {
	s := newTestState(t)
	s.AddProgram(1)
	s.AddProgram(1)
	s.AddProgram(1)
	initialData := s.Resources.Data

	s.applyUpkeep()

	expected := initialData - 3*s.Config.ProgramUpkeep
	if s.Resources.Data != expected {
		t.Errorf("upkeep: expected data=%d, got %d", expected, s.Resources.Data)
	}
}

func TestUpkeepStarvesProgram(t *testing.T) {
	s := newTestState(t)
	s.AddProgram(1)
	s.AddProgram(1)
	s.Resources.Data = 0

	s.applyUpkeep()

	if s.Resources.Data != 0 {
		t.Error("data should be clamped to 0")
	}
	if len(s.Programs) != 1 {
		t.Errorf("one program should starve, expected 1 remaining, got %d", len(s.Programs))
	}
}

func TestCoreHoldCostDeductsCompute(t *testing.T) {
	s := newTestState(t)
	s.AddProgram(4) // core node
	initialCompute := s.Resources.Compute

	s.applyCoreHoldCost()

	expected := initialCompute - s.Config.CoreHoldCost
	if s.Resources.Compute != expected {
		t.Errorf("core hold: expected compute=%d, got %d", expected, s.Resources.Compute)
	}
}

func TestCoreHoldCostEjectsProgram(t *testing.T) {
	s := newTestState(t)
	s.AddProgram(4) // core node
	s.Resources.Compute = 0

	s.applyCoreHoldCost()

	if s.Resources.Compute != 0 {
		t.Error("compute should be clamped to 0")
	}
	corePrograms := 0
	for _, p := range s.Programs {
		if p.NodeID == 4 {
			corePrograms++
		}
	}
	if corePrograms != 0 {
		t.Error("program should be ejected from core when compute is insufficient")
	}
}

// --- Win/Lose conditions ---

func TestWinCondition(t *testing.T) {
	cfg := testConfig()
	cfg.CoreWinThreshold = 2
	cfg.CoreWinDuration = 3

	net := network.New()
	net.AddNode(NewTestNode(1, network.NodeServer))
	net.AddNode(NewTestNode(2, network.NodeCore))
	net.Connect(1, 2)

	s := NewState(net, &cfg)
	// Put enough programs on core and server for mutual support
	s.AddProgram(2)
	s.AddProgram(2)
	s.AddProgram(1) // neighbor support

	s.Resources.Compute = 1000 // enough for core hold cost

	// Simulate ticks until win
	for i := range 5 {
		s.Tick++
		s.processEconomy()
		s.checkEndConditions()
		if s.GameOver && s.Won {
			if s.CoreHoldLen < cfg.CoreWinDuration {
				t.Errorf("won too early: coreHoldLen=%d, need %d", s.CoreHoldLen, cfg.CoreWinDuration)
			}
			return
		}
		_ = i
	}

	if !s.Won {
		t.Errorf("expected win after holding core for %d ticks, coreHoldLen=%d", cfg.CoreWinDuration, s.CoreHoldLen)
	}
}

func TestLoseCondition(t *testing.T) {
	s := newTestState(t)
	p := s.AddProgram(1)
	s.Tick = 10 // past grace period

	s.RemoveEntity(p.ID)
	s.checkEndConditions()

	if !s.GameOver {
		t.Error("game should be over when all programs destroyed")
	}
	if s.Won {
		t.Error("player should not win when all programs destroyed")
	}
}

func TestGracePeriodPreventsEarlyLoss(t *testing.T) {
	s := newTestState(t)
	p := s.AddProgram(1)
	s.Tick = 3 // within grace period

	s.RemoveEntity(p.ID)
	s.checkEndConditions()

	if s.GameOver {
		t.Error("game should NOT end during grace period (tick <= 5)")
	}
}

func TestCoreHoldResets(t *testing.T) {
	cfg := testConfig()
	cfg.CoreWinThreshold = 2
	cfg.CoreWinDuration = 5

	net := network.New()
	net.AddNode(NewTestNode(1, network.NodeServer))
	net.AddNode(NewTestNode(2, network.NodeCore))
	net.Connect(1, 2)

	s := NewState(net, &cfg)
	s.AddProgram(2)
	s.AddProgram(2)
	s.AddProgram(1)
	s.Resources.Compute = 1000

	// Hold for 2 ticks
	s.Tick++
	s.checkEndConditions()
	s.Tick++
	s.checkEndConditions()

	if s.CoreHoldLen != 2 {
		t.Fatalf("expected coreHoldLen=2, got %d", s.CoreHoldLen)
	}

	// Remove programs from core -> hold resets
	for id, p := range s.Programs {
		if p.NodeID == 2 {
			s.RemoveEntity(id)
		}
	}
	s.Tick++
	s.checkEndConditions()

	if s.CoreHoldLen != 0 {
		t.Errorf("core hold should reset when programs leave, got coreHoldLen=%d", s.CoreHoldLen)
	}
}

// --- Virus aging ---

func TestVirusExpires(t *testing.T) {
	s := newTestState(t)
	v := s.AddVirus(1)

	for range s.Config.VirusLifespan {
		s.ageViruses()
	}

	if _, ok := s.Viruses[v.ID]; ok {
		t.Error("virus should expire after lifespan ticks")
	}
}

func TestVirusSurvivesBeforeExpiry(t *testing.T) {
	s := newTestState(t)
	v := s.AddVirus(1)

	for range s.Config.VirusLifespan - 1 {
		s.ageViruses()
	}

	if _, ok := s.Viruses[v.ID]; !ok {
		t.Error("virus should still be alive before lifespan ends")
	}
}

// --- Multi-tick stability (regression test) ---

func TestStableColonyDoesNotCollapse(t *testing.T) {
	// Create a small stable network: 3 programs on connected servers
	// with mutual support. No ICE. Should survive indefinitely.
	cfg := testConfig()
	cfg.InitialData = 10000 // plenty of resources
	cfg.ICESpawnTick = 999
	cfg.ICEEscalationTick = 999

	net := network.New()
	net.AddNode(NewTestNode(1, network.NodeServer))
	net.AddNode(NewTestNode(2, network.NodeServer))
	net.AddNode(NewTestNode(3, network.NodeVault))
	net.Connect(1, 2)
	net.Connect(2, 3)
	net.Connect(1, 3)

	s := NewState(net, &cfg)
	s.rng = nil
	// Place programs with mutual support
	s.AddProgram(1)
	s.AddProgram(2)
	s.AddProgram(3)

	// Simulate 50 ticks — colony should remain stable
	for i := range 50 {
		s.Tick++
		s.applyRules()
		s.processEconomy()

		if len(s.Programs) == 0 {
			t.Fatalf("stable colony collapsed at tick %d — all programs dead", i+1)
		}
	}

	if len(s.Programs) < 3 {
		t.Errorf("expected at least 3 programs after 50 ticks, got %d", len(s.Programs))
	}
}

func TestAutoSpreadDoesNotCreateOvercrowdedPrograms(t *testing.T) {
	// Star topology: 11 nodes connected to center. Programs on all outer nodes.
	// Center should NOT get auto-spread (11 neighbors > SurviveMax 10).
	cfg := testConfig()

	net := network.New()
	net.AddNode(NewTestNode(20, network.NodeServer)) // center
	for i := uint64(1); i <= 11; i++ {
		net.AddNode(NewTestNode(i, network.NodeServer))
		net.Connect(20, i)
	}
	// Connect outer nodes for mutual support
	for i := uint64(1); i <= 11; i++ {
		for j := i + 1; j <= 11; j++ {
			net.Connect(i, j)
		}
	}

	s := NewState(net, &cfg)
	for i := uint64(1); i <= 11; i++ {
		s.AddProgram(i)
	}

	s.Tick++
	s.applyRules()

	// Center node (20) should NOT have a program
	for _, p := range s.Programs {
		if p.NodeID == 20 {
			t.Error("auto-spread should NOT place program on overcrowded node (11 neighbors > SurviveMax 10)")
		}
	}
}

// TestInitGameSurvives20Ticks verifies that a randomly generated game doesn't
// collapse within 20 ticks (no player intervention, just Conway rules + ICE).
// Run multiple times for statistical confidence.
func TestInitGameSurvives20Ticks(t *testing.T) {
	survivals := 0
	const runs = 20
	const ticks = 20

	for run := range runs {
		cfg := testConfig()
		cfg.ICESpawnTick = 999 // no new ICE
		cfg.ICEEscalationTick = 999
		state := InitGame(&cfg)

		for tick := range ticks {
			state.Tick++
			state.applyRules()
			state.processEconomy()
			state.checkEndConditions()
			if state.GameOver {
				t.Logf("run %d: game over at tick %d (programs=%d)", run, tick+1, len(state.Programs))
				break
			}
		}
		if !state.GameOver {
			survivals++
		}
	}

	// With balanced settings, at least 75% of games should survive 20 ticks
	// even without player input (just initial cluster + auto-spread)
	minSurvivals := runs * 3 / 4
	if survivals < minSurvivals {
		t.Errorf("only %d/%d games survived 20 ticks (need at least %d); balance is too harsh",
			survivals, runs, minSurvivals)
	}
	t.Logf("%d/%d games survived 20 ticks without player input", survivals, runs)
}
