package game

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand/v2"

	"github.com/dshlychkou/cyberspace/internal/entity"
	"github.com/dshlychkou/cyberspace/internal/network"
	"github.com/dshlychkou/cyberspace/internal/scheduler"
)

// TickCmd advances the game by one tick. The tick pipeline runs in order:
// scheduled events → virus aging → Conway rules → economy → end conditions.
// Calls OnComplete with a snapshot when done (including when paused/game over).
type TickCmd struct {
	OnComplete func(StateSnapshot)
}

func (t *TickCmd) Execute(_ context.Context, s *State) {
	if s.Paused || s.GameOver {
		if t.OnComplete != nil {
			t.OnComplete(s.Snapshot())
		}
		return
	}

	s.Tick++
	s.processScheduledEvents()
	s.ageViruses()
	s.applyRules()
	s.processEconomy()
	s.checkEndConditions()

	s.Score += len(s.Programs)
	s.Resources.Cycles = s.Tick

	if t.OnComplete != nil {
		t.OnComplete(s.Snapshot())
	}
}

// processScheduledEvents drains the scheduler for the current tick and
// dispatches each due event to its handler (ICE spawn or escalation).
func (s *State) processScheduledEvents() {
	for _, evt := range s.sched.DueEvents(s.Tick) {
		switch evt.Type {
		case scheduler.EventICESpawn:
			s.spawnScheduledICE()
		case scheduler.EventICEEscalation:
			s.escalateICE()
		}
	}
}

// spawnScheduledICE places one ICE on a random firewall or server and
// re-schedules the next spawn. The interval shrinks as ticks increase
// (min 3 ticks), ramping up pressure over time.
func (s *State) spawnScheduledICE() {
	candidates := s.Network.NodesByType(network.NodeFirewall)
	candidates = append(candidates, s.Network.NodesByType(network.NodeServer)...)
	if len(candidates) > 0 {
		target := candidates[s.rng.IntN(len(candidates))]
		s.AddICE(target.ID)
		s.AddEvent(fmt.Sprintf("New ICE spawned at %s", target.Label))
	}
	interval := max(s.Config.ICESpawnMinInterval, s.Config.ICESpawnTick-s.Tick/s.Config.ICESpawnTick)
	s.sched.Schedule(scheduler.Event{
		Tick:     s.Tick + interval,
		Priority: 1,
		Type:     scheduler.EventICESpawn,
	})
}

// escalateICE reinforces all firewalls and core nodes with new ICE in a
// single burst, then schedules the next escalation 40 ticks later.
func (s *State) escalateICE() {
	for _, fw := range s.Network.NodesByType(network.NodeFirewall) {
		s.AddICE(fw.ID)
	}
	for _, core := range s.Network.NodesByType(network.NodeCore) {
		s.AddICE(core.ID)
	}
	s.AddEvent("ICE defenses escalating — firewalls and core reinforced!")
	s.sched.Schedule(scheduler.Event{
		Tick:     s.Tick + s.Config.ICEEscalationRate,
		Priority: 0,
		Type:     scheduler.EventICEEscalation,
	})
}

// ageViruses increments each virus's age and removes any that exceeded
// their lifespan.
func (s *State) ageViruses() {
	for id, v := range s.Viruses {
		v.Tick()
		if !v.Alive {
			node := s.Network.GetNode(v.NodeID)
			s.AddEvent(fmt.Sprintf("Virus expired at %s (lifespan ended)", node.Label))
			s.RemoveEntity(id)
		}
	}
}

// applyRules evaluates Conway-style cellular automata rules across the
// network and applies the resulting deaths, spawns, moves, and flips.
func (s *State) applyRules() {
	snapshots := s.EntitySnapshots()
	ruleCfg := network.RuleConfig{
		SurviveMin:  s.Config.SurviveMin,
		SurviveMax:  s.Config.SurviveMax,
		SpreadExact: s.Config.SpreadExact,
	}
	result := network.EvaluateRules(s.Network, snapshots, ruleCfg)

	s.applyDeaths(result.Deaths)
	s.applySpawns(result.Spawns)
	s.applyMoves(result.Moves)
	s.applyFlips(result.Flips)
}

// applyDeaths removes killed entities and logs the cause. Programs die from
// ICE outnumbering them on the same node or from insufficient neighbor support.
func (s *State) applyDeaths(deaths []int) {
	for _, id := range deaths {
		if p, ok := s.Programs[id]; ok {
			node := s.Network.GetNode(p.NodeID)
			nodeICECount := 0
			nodeProgramCount := 0
			for _, ice := range s.ICEs {
				if ice.NodeID == p.NodeID {
					nodeICECount++
				}
			}
			for _, prog := range s.Programs {
				if prog.NodeID == p.NodeID {
					nodeProgramCount++
				}
			}
			if nodeICECount > nodeProgramCount {
				s.AddEvent(fmt.Sprintf("Program killed at %s (ICE outnumbers: %dI vs %dP)",
					node.Label, nodeICECount, nodeProgramCount))
			} else {
				s.AddEvent(fmt.Sprintf("Program died at %s (not enough nearby support)",
					node.Label))
			}
		}
		if ice, ok := s.ICEs[id]; ok {
			node := s.Network.GetNode(ice.NodeID)
			s.AddEvent(fmt.Sprintf("ICE destroyed at %s", node.Label))
		}
		s.RemoveEntity(id)
	}
}

// applySpawns creates new entities from Conway auto-spread results.
// Programs spread to empty nodes when they have enough neighbors.
func (s *State) applySpawns(spawns []network.SpawnAction) {
	for _, spawn := range spawns {
		if spawn.Kind == entity.KindProgram {
			node := s.Network.GetNode(spawn.NodeID)
			s.AddProgram(spawn.NodeID)
			s.AddEvent(fmt.Sprintf("Program auto-spread to %s (%d nearby programs)",
				node.Label, spawn.NeighborCnt))
		}
	}
}

// applyMoves relocates ICE entities that are patrolling toward nodes with
// programs. Each entity moves at most once per tick (dedup by entity ID).
func (s *State) applyMoves(moves []network.MoveAction) {
	moved := make(map[int]bool)
	for _, move := range moves {
		if moved[move.EntityID] {
			continue
		}
		moved[move.EntityID] = true
		fromNode := ""
		if ice, ok := s.ICEs[move.EntityID]; ok {
			if n := s.Network.GetNode(ice.NodeID); n != nil {
				fromNode = n.Label
			}
		}
		s.MoveEntity(move.EntityID, move.ToNodeID)
		toNode := s.Network.GetNode(move.ToNodeID)
		if fromNode != "" && toNode != nil {
			s.AddEvent(fmt.Sprintf("ICE patrolled %s → %s (hunting programs)",
				fromNode, toNode.Label))
		}
	}
}

// applyFlips converts ICE entities into programs when a virus on an
// adjacent node corrupts them.
func (s *State) applyFlips(flips []network.FlipAction) {
	for _, flip := range flips {
		if _, ok := s.ICEs[flip.EntityID]; ok {
			node := s.Network.GetNode(flip.NodeID)
			s.FlipICEToProgram(flip.EntityID)
			s.AddEvent(fmt.Sprintf("Virus corrupted ICE → Program at %s", node.Label))
		}
	}
}

// processEconomy runs the three economy phases in order: harvest income
// from vaults/relays, deduct program upkeep, then deduct core hold cost.
func (s *State) processEconomy() {
	s.harvestResources()
	s.applyUpkeep()
	s.applyCoreHoldCost()
}

// harvestResources grants Data for each program on a vault (+DataHarvestRate)
// and Compute for each program on a relay (+2).
func (s *State) harvestResources() {
	for _, p := range s.Programs {
		node := s.Network.GetNode(p.NodeID)
		if node == nil {
			continue
		}
		switch node.Type {
		case network.NodeVault:
			s.Resources.Data += s.Config.DataHarvestRate
			s.Score += s.Config.DataHarvestRate
		case network.NodeRelay:
			s.Resources.Compute += s.Config.ComputeHarvestRate
		}
	}
}

// applyUpkeep deducts Data for each living program. If Data goes negative,
// it's clamped to zero and one random program starves per tick.
func (s *State) applyUpkeep() {
	upkeep := len(s.Programs) * s.Config.ProgramUpkeep
	s.Resources.Data -= upkeep
	if s.Resources.Data < 0 {
		s.Resources.Data = 0
		for id, p := range s.Programs {
			node := s.Network.GetNode(p.NodeID)
			s.RemoveEntity(id)
			s.AddEvent(fmt.Sprintf("Program starved at %s (no Data)", node.Label))
			break
		}
	}
}

// applyCoreHoldCost deducts Compute for each program occupying a core node.
// If Compute goes negative, one program is ejected from the core.
func (s *State) applyCoreHoldCost() {
	coreNodes := s.Network.NodesByType(network.NodeCore)
	programsOnCore := s.countProgramsOnCore(coreNodes)

	coreCost := programsOnCore * s.Config.CoreHoldCost
	s.Resources.Compute -= coreCost
	if s.Resources.Compute < 0 {
		s.Resources.Compute = 0
		if programsOnCore > 0 && len(coreNodes) > 0 {
			for _, eid := range coreNodes[0].Entities {
				if _, ok := s.Programs[eid]; ok {
					s.RemoveEntity(eid)
					s.AddEvent("Program on CORE failed (no Compute)")
					break
				}
			}
		}
	}
}

// checkEndConditions evaluates win/lose. Win: hold >= CoreWinThreshold
// programs on core for CoreWinDuration consecutive ticks. Lose: all
// programs destroyed after tick 5.
func (s *State) checkEndConditions() {
	coreNodes := s.Network.NodesByType(network.NodeCore)
	programsOnCore := s.countProgramsOnCore(coreNodes)

	if programsOnCore >= s.Config.CoreWinThreshold {
		s.CoreHoldLen++
		if s.CoreHoldLen >= s.Config.CoreWinDuration {
			s.GameOver = true
			s.Won = true
			s.AddEvent("CORE CAPTURED! You win!")
		} else {
			s.AddEvent(fmt.Sprintf("Holding CORE: %d/%d ticks (%d programs on core)",
				s.CoreHoldLen, s.Config.CoreWinDuration, programsOnCore))
		}
	} else {
		if s.CoreHoldLen > 0 {
			s.AddEvent(fmt.Sprintf("CORE hold lost! Programs on core: %d (need %d)",
				programsOnCore, s.Config.CoreWinThreshold))
		}
		s.CoreHoldLen = 0
	}

	const gracePeriod = 5 // don't end the game during initial setup ticks
	if len(s.Programs) == 0 && s.Tick > gracePeriod {
		s.GameOver = true
		s.Won = false
		s.AddEvent("All programs destroyed. Game over.")
	}
}

// countProgramsOnCore returns the total number of player programs across
// all core nodes.
func (s *State) countProgramsOnCore(coreNodes []*network.Node) int {
	count := 0
	for _, core := range coreNodes {
		for _, eid := range core.Entities {
			if _, ok := s.Programs[eid]; ok {
				count++
			}
		}
	}
	return count
}

// spawnEntity is a common helper for resource-gated entity placement commands.
func spawnEntity(
	s *State, nodeID uint64, resource *int, cost int,
	resourceName string, place func(uint64), eventMsg string,
	onComplete func(bool, string),
) {
	if *resource < cost {
		if onComplete != nil {
			onComplete(false, fmt.Sprintf("Not enough %s (need %d, have %d)", resourceName, cost, *resource))
		}
		return
	}
	node := s.Network.GetNode(nodeID)
	if node == nil {
		if onComplete != nil {
			onComplete(false, "Invalid node")
		}
		return
	}
	*resource -= cost
	place(nodeID)
	s.AddEvent(fmt.Sprintf(eventMsg, node.Label, cost))
	if onComplete != nil {
		onComplete(true, "")
	}
}

// SpawnProgramCmd places a new program on the given node if the player
// has enough Data. Deducts ProgramSpawnCost on success.
type SpawnProgramCmd struct {
	NodeID     uint64
	OnComplete func(bool, string)
}

func (c *SpawnProgramCmd) Execute(_ context.Context, s *State) {
	spawnEntity(s, c.NodeID, &s.Resources.Data, s.Config.ProgramSpawnCost, "Data",
		func(id uint64) { s.AddProgram(id) }, "You spawned a program at %s (-%d Data)", c.OnComplete)
}

// DeployVirusCmd places a virus on the given node if the player has enough
// Compute. The virus corrupts adjacent ICE, converting them into programs.
type DeployVirusCmd struct {
	NodeID     uint64
	OnComplete func(bool, string)
}

func (c *DeployVirusCmd) Execute(_ context.Context, s *State) {
	spawnEntity(s, c.NodeID, &s.Resources.Compute, s.Config.VirusDeployCost, "Compute",
		func(id uint64) { s.AddVirus(id) }, "You deployed a virus at %s (-%d Compute, corrupts adjacent ICE)", c.OnComplete)
}

// TogglePauseCmd flips the paused flag. While paused, TickCmd still
// returns a snapshot but skips all simulation.
type TogglePauseCmd struct{}

func (c *TogglePauseCmd) Execute(_ context.Context, s *State) {
	s.Paused = !s.Paused
}

// GetStateCmd returns a snapshot of the current state without advancing
// the simulation. Used by the TUI to fetch state on demand.
type GetStateCmd struct {
	OnComplete func(StateSnapshot)
}

func (c *GetStateCmd) Execute(_ context.Context, s *State) {
	if c.OnComplete != nil {
		c.OnComplete(s.Snapshot())
	}
}

// InitGame creates a fully initialized game state: generates the network,
// places initial programs on one server (clustered for mutual support),
// places ICE on firewalls and core, and schedules recurring ICE events.
func InitGame(cfg *Config) *State {
	var seed [16]byte
	_, _ = crand.Read(seed[:])
	s1 := binary.LittleEndian.Uint64(seed[:8])
	s2 := binary.LittleEndian.Uint64(seed[8:])
	rng := rand.New(rand.NewPCG(s1, s2)) //nolint:gosec // seeded from crypto/rand
	net := network.Generate(rng)

	s := NewState(net, cfg)
	s.rng = rng
	s.sched = scheduler.New()

	// Spawn initial programs clustered on one server (mutual support)
	servers := net.NodesByType(network.NodeServer)
	if len(servers) > 0 {
		start := servers[0]
		for range cfg.InitialPrograms {
			s.AddProgram(start.ID)
		}
		s.AddEvent(fmt.Sprintf("%d initial programs at %s", cfg.InitialPrograms, start.Label))
	}

	// Spawn initial ICE on firewalls
	firewalls := net.NodesByType(network.NodeFirewall)
	for i := range min(cfg.InitialICE, len(firewalls)) {
		s.AddICE(firewalls[i].ID)
		s.AddEvent(fmt.Sprintf("Initial ICE at %s (blocks path to core)", firewalls[i].Label))
	}

	// Defend core from the start
	coreNodes := net.NodesByType(network.NodeCore)
	for _, core := range coreNodes {
		s.AddICE(core.ID)
		s.AddEvent(fmt.Sprintf("Initial ICE at %s (core defense)", core.Label))
	}

	// Schedule ICE escalation
	s.sched.Schedule(scheduler.Event{
		Tick:     cfg.ICESpawnTick,
		Priority: 1,
		Type:     scheduler.EventICESpawn,
	})
	s.sched.Schedule(scheduler.Event{
		Tick:     cfg.ICEEscalationTick,
		Priority: 0,
		Type:     scheduler.EventICEEscalation,
	})

	return s
}
