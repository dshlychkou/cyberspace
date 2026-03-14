package game

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/barnowlsnest/go-logslib/v2/pkg/logger"

	"github.com/dshlychkou/cyberspace/internal/entity"
	"github.com/dshlychkou/cyberspace/internal/network"
	"github.com/dshlychkou/cyberspace/internal/scheduler"
)

func newGameRNG() (*rand.Rand, error) {
	var seed [16]byte
	if _, err := crand.Read(seed[:]); err != nil {
		return nil, fmt.Errorf("seed RNG: %w", err)
	}
	s1 := binary.LittleEndian.Uint64(seed[:8])
	s2 := binary.LittleEndian.Uint64(seed[8:])
	return rand.New(rand.NewPCG(s1, s2)), nil
}

const maxStoredEvents = 100

type State struct {
	Network      *network.Network
	Config       Config
	Programs     map[int]*entity.Program
	ICEs         map[int]*entity.ICE
	Viruses      map[int]*entity.Virus
	Resources    Resources
	Events       []Event
	sched        *scheduler.Scheduler
	rng          *rand.Rand
	eventLogger  *logger.Logger
	eventLogFile *os.File
	Tick         int
	CoreHoldLen  int // consecutive ticks with enough programs on core
	Score        int
	nextEntityID int
	Paused       bool
	GameOver     bool
	Won          bool
}

type Resources struct {
	Data    int `json:"data"`
	Compute int `json:"compute"`
	Cycles  int `json:"cycles"`
}

type Event struct {
	Message string `json:"message"`
	Tick    int    `json:"tick"`
}

type StateSnapshot struct {
	Resources        Resources
	Programs         []ProgramSnapshot
	ICEs             []ICESnapshot
	Viruses          []VirusSnapshot
	Nodes            map[uint64]NodeSnapshot
	Edges            []EdgeSnapshot
	Events           []Event
	Tick             int
	Score            int
	CoreHoldLen      int
	CoreWinThreshold int
	CoreWinDuration  int
	ProgramSpawnCost int
	VirusDeployCost  int
	DataIncome       int
	DataBurn         int
	ComputeIncome    int
	ComputeBurn      int
	Paused           bool
	GameOver         bool
	Won              bool
}

type ProgramSnapshot struct {
	NodeID uint64
	ID     int
	State  entity.ProgramState
}

type ICESnapshot struct {
	NodeID uint64
	ID     int
	State  entity.ICEState
}

type VirusSnapshot struct {
	NodeID uint64
	ID     int
	Age    int
	MaxAge int
}

type NodeSnapshot struct {
	Label    string
	Entities []int
	ID       uint64
	Type     network.NodeType
}

type EdgeSnapshot struct {
	From uint64
	To   uint64
}

func NewState(net *network.Network, cfg *Config) *State {
	return &State{
		Network:      net,
		Config:       *cfg,
		Programs:     make(map[int]*entity.Program),
		ICEs:         make(map[int]*entity.ICE),
		Viruses:      make(map[int]*entity.Virus),
		Resources:    Resources{Data: cfg.InitialData, Compute: cfg.InitialCompute, Cycles: 0},
		nextEntityID: 1,
	}
}

func (s *State) IsProvidable() bool { return s != nil }

func (s *State) NextID() int {
	id := s.nextEntityID
	s.nextEntityID++
	return id
}

func (s *State) AddProgram(nodeID uint64) *entity.Program {
	id := s.NextID()
	p := entity.NewProgram(id, nodeID)
	s.Programs[id] = p
	node := s.Network.GetNode(nodeID)
	if node != nil {
		node.AddEntity(id)
	}
	return p
}

func (s *State) AddICE(nodeID uint64) *entity.ICE {
	id := s.NextID()
	ice := entity.NewICE(id, nodeID)
	s.ICEs[id] = ice
	node := s.Network.GetNode(nodeID)
	if node != nil {
		node.AddEntity(id)
	}
	return ice
}

func (s *State) AddVirus(nodeID uint64) *entity.Virus {
	id := s.NextID()
	v := entity.NewVirus(id, nodeID, s.Config.VirusLifespan)
	s.Viruses[id] = v
	node := s.Network.GetNode(nodeID)
	if node != nil {
		node.AddEntity(id)
	}
	return v
}

func (s *State) RemoveEntity(id int) {
	if p, ok := s.Programs[id]; ok {
		if node := s.Network.GetNode(p.NodeID); node != nil {
			node.RemoveEntity(id)
		}
		delete(s.Programs, id)
		return
	}
	if ice, ok := s.ICEs[id]; ok {
		if node := s.Network.GetNode(ice.NodeID); node != nil {
			node.RemoveEntity(id)
		}
		delete(s.ICEs, id)
		return
	}
	if v, ok := s.Viruses[id]; ok {
		if node := s.Network.GetNode(v.NodeID); node != nil {
			node.RemoveEntity(id)
		}
		delete(s.Viruses, id)
		return
	}
}

func (s *State) MoveEntity(id int, toNodeID uint64) {
	if p, ok := s.Programs[id]; ok {
		if node := s.Network.GetNode(p.NodeID); node != nil {
			node.RemoveEntity(id)
		}
		p.NodeID = toNodeID
		if node := s.Network.GetNode(toNodeID); node != nil {
			node.AddEntity(id)
		}
		return
	}
	if ice, ok := s.ICEs[id]; ok {
		if node := s.Network.GetNode(ice.NodeID); node != nil {
			node.RemoveEntity(id)
		}
		ice.NodeID = toNodeID
		if node := s.Network.GetNode(toNodeID); node != nil {
			node.AddEntity(id)
		}
		return
	}
}

func (s *State) FlipICEToProgram(iceID int) {
	ice, ok := s.ICEs[iceID]
	if !ok {
		return
	}
	nodeID := ice.NodeID
	s.RemoveEntity(iceID)
	s.AddProgram(nodeID)
}

func (s *State) EntitySnapshots() []network.EntitySnapshot {
	snaps := make([]network.EntitySnapshot, 0, len(s.Programs)+len(s.ICEs)+len(s.Viruses))
	for _, p := range s.Programs {
		if p.Alive {
			snaps = append(snaps, network.EntitySnapshot{ID: p.ID, Kind: p.Kind, NodeID: p.NodeID})
		}
	}
	for _, ice := range s.ICEs {
		if ice.Alive {
			snaps = append(snaps, network.EntitySnapshot{ID: ice.ID, Kind: ice.Kind, NodeID: ice.NodeID})
		}
	}
	for _, v := range s.Viruses {
		if v.Alive {
			snaps = append(snaps, network.EntitySnapshot{ID: v.ID, Kind: v.Kind, NodeID: v.NodeID})
		}
	}
	return snaps
}

func (s *State) AddEvent(msg string) {
	s.Events = append(s.Events, Event{
		Tick:    s.Tick,
		Message: msg,
	})
	if len(s.Events) > maxStoredEvents {
		s.Events = s.Events[len(s.Events)-maxStoredEvents:]
	}
	if s.eventLogger != nil {
		s.eventLogger.Info(msg, logger.Field{Key: "tick", Value: s.Tick})
	}
}

// initEventLog opens the event log file and creates a logger for it.
func (s *State) initEventLog(path string) error {
	f, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open event log: %w", err)
	}
	s.eventLogFile = f
	s.eventLogger = logger.New(logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.JSONFormat,
		Output: f,
		UseUTC: true,
	})
	return nil
}

// CloseEventLog flushes and closes the event log file if open.
func (s *State) CloseEventLog() {
	if s.eventLogger != nil {
		s.eventLogger.Flush()
		s.eventLogger = nil
	}
	if s.eventLogFile != nil {
		_ = s.eventLogFile.Close()
		s.eventLogFile = nil
	}
}

func (s *State) Snapshot() StateSnapshot {
	snap := StateSnapshot{
		Tick:             s.Tick,
		Paused:           s.Paused,
		GameOver:         s.GameOver,
		Won:              s.Won,
		Resources:        s.Resources,
		Score:            s.Score,
		CoreHoldLen:      s.CoreHoldLen,
		CoreWinThreshold: s.Config.CoreWinThreshold,
		CoreWinDuration:  s.Config.CoreWinDuration,
		ProgramSpawnCost: s.Config.ProgramSpawnCost,
		VirusDeployCost:  s.Config.VirusDeployCost,
		Programs:         s.snapshotPrograms(),
		ICEs:             s.snapshotICEs(),
		Viruses:          s.snapshotViruses(),
		Nodes:            s.snapshotNodes(),
		Edges:            s.snapshotEdges(),
		Events:           s.snapshotEvents(),
	}
	s.computeEconomyRates(&snap)
	return snap
}

func (s *State) snapshotPrograms() []ProgramSnapshot {
	out := make([]ProgramSnapshot, 0, len(s.Programs))
	for _, p := range s.Programs {
		out = append(out, ProgramSnapshot{ID: p.ID, NodeID: p.NodeID, State: p.State})
	}
	return out
}

func (s *State) snapshotICEs() []ICESnapshot {
	out := make([]ICESnapshot, 0, len(s.ICEs))
	for _, ice := range s.ICEs {
		out = append(out, ICESnapshot{ID: ice.ID, NodeID: ice.NodeID, State: ice.State})
	}
	return out
}

func (s *State) snapshotViruses() []VirusSnapshot {
	out := make([]VirusSnapshot, 0, len(s.Viruses))
	for _, v := range s.Viruses {
		out = append(out, VirusSnapshot{ID: v.ID, NodeID: v.NodeID, Age: v.Age, MaxAge: v.MaxAge})
	}
	return out
}

func (s *State) snapshotNodes() map[uint64]NodeSnapshot {
	out := make(map[uint64]NodeSnapshot, len(s.Network.Nodes))
	for id, n := range s.Network.Nodes {
		out[id] = NodeSnapshot{
			ID:       n.ID,
			Type:     n.Type,
			Label:    n.Label,
			Entities: append([]int{}, n.Entities...),
		}
	}
	return out
}

func (s *State) snapshotEdges() []EdgeSnapshot {
	out := make([]EdgeSnapshot, 0)
	for _, nodeID := range s.Network.NodeIDs() {
		for _, neighborID := range s.Network.Neighbors(nodeID) {
			if nodeID < neighborID {
				out = append(out, EdgeSnapshot{From: nodeID, To: neighborID})
			}
		}
	}
	return out
}

func (s *State) snapshotEvents() []Event {
	eventStart := 0
	if len(s.Events) > s.Config.EventLogSize {
		eventStart = len(s.Events) - s.Config.EventLogSize
	}
	return append([]Event{}, s.Events[eventStart:]...)
}

func (s *State) computeEconomyRates(snap *StateSnapshot) {
	for _, p := range s.Programs {
		node := s.Network.GetNode(p.NodeID)
		if node != nil {
			switch node.Type {
			case network.NodeVault:
				snap.DataIncome += s.Config.DataHarvestRate
			case network.NodeRelay:
				snap.ComputeIncome += s.Config.ComputeHarvestRate
			}
		}
	}
	snap.DataBurn = len(s.Programs) * s.Config.ProgramUpkeep
	for _, core := range s.Network.NodesByType(network.NodeCore) {
		for _, eid := range core.Entities {
			if _, ok := s.Programs[eid]; ok {
				snap.ComputeBurn += s.Config.CoreHoldCost
			}
		}
	}
}
