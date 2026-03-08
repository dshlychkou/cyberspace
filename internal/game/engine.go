package game

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/dshlychkou/cyberspace/internal/entity"
	"github.com/dshlychkou/cyberspace/internal/network"
	"github.com/dshlychkou/cyberspace/internal/scheduler"
)

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

	// Process scheduled events
	for _, evt := range s.sched.DueEvents(s.Tick) {
		switch evt.Type {
		case scheduler.EventICESpawn:
			candidates := s.Network.NodesByType(network.NodeFirewall)
			candidates = append(candidates, s.Network.NodesByType(network.NodeServer)...)
			if len(candidates) > 0 {
				target := candidates[s.rng.IntN(len(candidates))]
				s.AddICE(uint64(target.ID))
				s.AddEvent(fmt.Sprintf("New ICE spawned at %s", target.Label))
			}
			// Schedule next recurring spawn
			interval := max(3, 12-s.Tick/10)
			s.sched.Schedule(scheduler.Event{
				Tick:     s.Tick + interval,
				Priority: 1,
				Type:     scheduler.EventICESpawn,
			})
		case scheduler.EventICEEscalation:
			// Spawn a burst of ICE across firewalls and core
			fwNodes := s.Network.NodesByType(network.NodeFirewall)
			for _, fw := range fwNodes {
				s.AddICE(uint64(fw.ID))
			}
			coreNodes := s.Network.NodesByType(network.NodeCore)
			for _, core := range coreNodes {
				s.AddICE(uint64(core.ID))
			}
			s.AddEvent("ICE defenses escalating — firewalls and core reinforced!")
			// Schedule next escalation
			s.sched.Schedule(scheduler.Event{
				Tick:     s.Tick + 40,
				Priority: 0,
				Type:     scheduler.EventICEEscalation,
			})
		}
	}

	// Age viruses
	for id, v := range s.Viruses {
		v.Tick()
		if !v.Alive {
			node := s.Network.GetNode(v.NodeID)
			s.AddEvent(fmt.Sprintf("Virus expired at %s (lifespan ended)", node.Label))
			s.RemoveEntity(id)
		}
	}

	// Evaluate Conway rules
	snapshots := s.EntitySnapshots()
	ruleCfg := network.RuleConfig{
		SurviveMin:  s.Config.SurviveMin,
		SurviveMax:  s.Config.SurviveMax,
		SpreadExact: s.Config.SpreadExact,
	}
	result := network.EvaluateRules(s.Network, snapshots, ruleCfg)

	// Apply deaths with explanations
	for _, id := range result.Deaths {
		if p, ok := s.Programs[id]; ok {
			node := s.Network.GetNode(p.NodeID)
			// Determine reason
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

	// Apply spawns with explanations
	for _, spawn := range result.Spawns {
		node := s.Network.GetNode(spawn.NodeID)
		switch spawn.Kind {
		case entity.KindProgram:
			s.AddProgram(spawn.NodeID)
			s.AddEvent(fmt.Sprintf("Program auto-spread to %s (%d nearby programs)",
				node.Label, spawn.NeighborCnt))
		}
	}

	// Apply moves
	moved := make(map[int]bool)
	for _, move := range result.Moves {
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

	// Apply flips (virus converts ICE)
	for _, flip := range result.Flips {
		if _, ok := s.ICEs[flip.EntityID]; ok {
			node := s.Network.GetNode(flip.NodeID)
			s.FlipICEToProgram(flip.EntityID)
			s.AddEvent(fmt.Sprintf("Virus corrupted ICE → Program at %s", node.Label))
		}
	}

	// Harvest resources from data nodes
	for _, p := range s.Programs {
		node := s.Network.GetNode(p.NodeID)
		if node != nil && node.Type == network.NodeVault {
			s.Resources.Data += s.Config.DataHarvestRate
			s.Score += s.Config.DataHarvestRate
		}
	}

	// Compute from relays
	for _, p := range s.Programs {
		node := s.Network.GetNode(p.NodeID)
		if node != nil && node.Type == network.NodeRelay {
			s.Resources.Compute += 2
		}
	}

	// Program upkeep: each program costs Data per tick
	upkeep := len(s.Programs) * s.Config.ProgramUpkeep
	s.Resources.Data -= upkeep
	if s.Resources.Data < 0 {
		s.Resources.Data = 0
		// Starvation: kill one random program per tick while bankrupt
		for id, p := range s.Programs {
			node := s.Network.GetNode(p.NodeID)
			s.RemoveEntity(id)
			s.AddEvent(fmt.Sprintf("Program starved at %s (no Data)", node.Label))
			break
		}
	}

	// Core hold cost: programs on core drain Compute
	coreNodes := s.Network.NodesByType(network.NodeCore)
	programsOnCore := 0
	for _, core := range coreNodes {
		for _, eid := range core.Entities {
			if _, ok := s.Programs[eid]; ok {
				programsOnCore++
			}
		}
	}
	coreCost := programsOnCore * s.Config.CoreHoldCost
	s.Resources.Compute -= coreCost
	if s.Resources.Compute < 0 {
		s.Resources.Compute = 0
		// Kill one program on core if can't sustain
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

	// Check win condition: programs on core
	// (recount after possible core starvation)
	programsOnCore = 0
	for _, core := range coreNodes {
		for _, eid := range core.Entities {
			if _, ok := s.Programs[eid]; ok {
				programsOnCore++
			}
		}
	}

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

	// Check lose condition: no programs left
	if len(s.Programs) == 0 && s.Tick > 5 {
		s.GameOver = true
		s.Won = false
		s.AddEvent("All programs destroyed. Game over.")
	}

	s.Score += len(s.Programs)
	s.Resources.Cycles = s.Tick

	if t.OnComplete != nil {
		t.OnComplete(s.Snapshot())
	}
}

type SpawnProgramCmd struct {
	NodeID     uint64
	OnComplete func(bool, string)
}

func (c *SpawnProgramCmd) Execute(_ context.Context, s *State) {
	if s.Resources.Data < s.Config.ProgramSpawnCost {
		if c.OnComplete != nil {
			c.OnComplete(false, fmt.Sprintf("Not enough Data (need %d, have %d)",
				s.Config.ProgramSpawnCost, s.Resources.Data))
		}
		return
	}
	node := s.Network.GetNode(c.NodeID)
	if node == nil {
		if c.OnComplete != nil {
			c.OnComplete(false, "Invalid node")
		}
		return
	}
	s.Resources.Data -= s.Config.ProgramSpawnCost
	s.AddProgram(c.NodeID)
	s.AddEvent(fmt.Sprintf("You spawned a program at %s (-%d Data)",
		node.Label, s.Config.ProgramSpawnCost))
	if c.OnComplete != nil {
		c.OnComplete(true, "")
	}
}

type DeployVirusCmd struct {
	NodeID     uint64
	OnComplete func(bool, string)
}

func (c *DeployVirusCmd) Execute(_ context.Context, s *State) {
	if s.Resources.Compute < s.Config.VirusDeployCost {
		if c.OnComplete != nil {
			c.OnComplete(false, fmt.Sprintf("Not enough Compute (need %d, have %d)",
				s.Config.VirusDeployCost, s.Resources.Compute))
		}
		return
	}
	node := s.Network.GetNode(c.NodeID)
	if node == nil {
		if c.OnComplete != nil {
			c.OnComplete(false, "Invalid node")
		}
		return
	}
	s.Resources.Compute -= s.Config.VirusDeployCost
	s.AddVirus(c.NodeID)
	s.AddEvent(fmt.Sprintf("You deployed a virus at %s (-%d Compute, corrupts adjacent ICE)",
		node.Label, s.Config.VirusDeployCost))
	if c.OnComplete != nil {
		c.OnComplete(true, "")
	}
}

type TogglePauseCmd struct{}

func (c *TogglePauseCmd) Execute(_ context.Context, s *State) {
	s.Paused = !s.Paused
}

type GetStateCmd struct {
	OnComplete func(StateSnapshot)
}

func (c *GetStateCmd) Execute(_ context.Context, s *State) {
	if c.OnComplete != nil {
		c.OnComplete(s.Snapshot())
	}
}

func InitGame(cfg Config) *State {
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	net := network.Generate(rng)

	s := NewState(net, cfg)
	s.rng = rng
	s.sched = scheduler.New()

	// Spawn initial programs clustered on one server (mutual support)
	servers := net.NodesByType(network.NodeServer)
	if len(servers) > 0 {
		start := servers[0]
		for range cfg.InitialPrograms {
			s.AddProgram(uint64(start.ID))
		}
		s.AddEvent(fmt.Sprintf("%d initial programs at %s", cfg.InitialPrograms, start.Label))
	}

	// Spawn initial ICE on firewalls
	firewalls := net.NodesByType(network.NodeFirewall)
	for i := range min(cfg.InitialICE, len(firewalls)) {
		s.AddICE(uint64(firewalls[i].ID))
		s.AddEvent(fmt.Sprintf("Initial ICE at %s (blocks path to core)", firewalls[i].Label))
	}

	// Defend core from the start
	coreNodes := net.NodesByType(network.NodeCore)
	for _, core := range coreNodes {
		s.AddICE(uint64(core.ID))
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
