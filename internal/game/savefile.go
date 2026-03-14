package game

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dshlychkou/cyberspace/internal/entity"
	"github.com/dshlychkou/cyberspace/internal/network"
	"github.com/dshlychkou/cyberspace/internal/scheduler"
)

type SaveFile struct {
	Version         int               `json:"version"`
	SavedAt         time.Time         `json:"saved_at"`
	Config          Config            `json:"config"`
	Tick            int               `json:"tick"`
	Score           int               `json:"score"`
	CoreHoldLen     int               `json:"core_hold_len"`
	NextEntityID    int               `json:"next_entity_id"`
	Paused          bool              `json:"paused"`
	GameOver        bool              `json:"game_over"`
	Won             bool              `json:"won"`
	Resources       Resources         `json:"resources"`
	Events          []Event           `json:"events"`
	Nodes           []SaveNode        `json:"nodes"`
	Edges           []SaveEdge        `json:"edges"`
	Programs        []*entity.Program `json:"programs"`
	ICEs            []*entity.ICE     `json:"ices"`
	Viruses         []*entity.Virus   `json:"viruses"`
	ScheduledEvents []scheduler.Event `json:"scheduled_events"`
}

type SaveNode struct {
	ID       uint64           `json:"id"`
	Type     network.NodeType `json:"type"`
	Label    string           `json:"label"`
	Entities []int            `json:"entities"`
}

type SaveEdge struct {
	From uint64 `json:"from"`
	To   uint64 `json:"to"`
}

type SaveFileInfo struct {
	Name    string
	Path    string
	ModTime time.Time
}

func (s *State) ToSaveFile() SaveFile {
	sf := SaveFile{
		Version:      1,
		SavedAt:      time.Now(),
		Config:       s.Config,
		Tick:         s.Tick,
		Score:        s.Score,
		CoreHoldLen:  s.CoreHoldLen,
		NextEntityID: s.nextEntityID,
		Paused:       s.Paused,
		GameOver:     s.GameOver,
		Won:          s.Won,
		Resources:    s.Resources,
		Events:       append([]Event{}, s.Events...),
	}

	// Nodes
	for _, n := range s.Network.Nodes {
		sf.Nodes = append(sf.Nodes, SaveNode{
			ID:       n.ID,
			Type:     n.Type,
			Label:    n.Label,
			Entities: append([]int{}, n.Entities...),
		})
	}
	sort.Slice(sf.Nodes, func(i, j int) bool { return sf.Nodes[i].ID < sf.Nodes[j].ID })

	// Edges (deduped: from < to)
	for _, nodeID := range s.Network.NodeIDs() {
		for _, neighborID := range s.Network.Neighbors(nodeID) {
			if nodeID < neighborID {
				sf.Edges = append(sf.Edges, SaveEdge{From: nodeID, To: neighborID})
			}
		}
	}

	// Entities
	for _, p := range s.Programs {
		sf.Programs = append(sf.Programs, p)
	}
	for _, ice := range s.ICEs {
		sf.ICEs = append(sf.ICEs, ice)
	}
	for _, v := range s.Viruses {
		sf.Viruses = append(sf.Viruses, v)
	}

	// Scheduler
	if s.sched != nil {
		sf.ScheduledEvents = s.sched.PendingEvents()
	}

	return sf
}

func FromSaveFile(sf *SaveFile) *State {
	net := network.New()
	for _, sn := range sf.Nodes {
		node := &network.Node{
			ID:       sn.ID,
			Type:     sn.Type,
			Label:    sn.Label,
			Entities: append([]int{}, sn.Entities...),
		}
		net.AddNode(node)
	}
	for _, e := range sf.Edges {
		net.Connect(e.From, e.To)
	}

	s := &State{
		Network:      net,
		Config:       sf.Config,
		Programs:     make(map[int]*entity.Program),
		ICEs:         make(map[int]*entity.ICE),
		Viruses:      make(map[int]*entity.Virus),
		Resources:    sf.Resources,
		Events:       append([]Event{}, sf.Events...),
		Tick:         sf.Tick,
		Score:        sf.Score,
		CoreHoldLen:  sf.CoreHoldLen,
		nextEntityID: sf.NextEntityID,
		Paused:       sf.Paused,
		GameOver:     sf.GameOver,
		Won:          sf.Won,
	}

	for _, p := range sf.Programs {
		s.Programs[p.ID] = p
	}
	for _, ice := range sf.ICEs {
		s.ICEs[ice.ID] = ice
	}
	for _, v := range sf.Viruses {
		s.Viruses[v.ID] = v
	}

	// Restore scheduler
	if len(sf.ScheduledEvents) > 0 {
		s.sched = scheduler.Restore(sf.ScheduledEvents)
	} else {
		s.sched = scheduler.New()
	}

	// Fresh RNG
	var seed [16]byte
	_, _ = crand.Read(seed[:])
	s1 := binary.LittleEndian.Uint64(seed[:8])
	s2 := binary.LittleEndian.Uint64(seed[8:])
	s.rng = rand.New(rand.NewPCG(s1, s2)) //nolint:gosec // seeded from crypto/rand

	// Init event log if configured
	if sf.Config.EventLogFile != "" {
		if err := s.initEventLog(sf.Config.EventLogFile); err != nil {
			s.AddEvent(fmt.Sprintf("Warning: could not open event log file: %v", err))
		}
	}

	return s
}

func ResolveSaveDir(configured string) (string, error) {
	dir := configured
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
		dir = filepath.Join(home, ".cyberspace", "saves")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create save dir: %w", err)
	}
	return dir, nil
}

func WriteSaveFile(path string, sf *SaveFile) error {
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal save: %w", err)
	}
	if err := os.WriteFile(filepath.Clean(path), data, 0o600); err != nil {
		return fmt.Errorf("write save: %w", err)
	}
	return nil
}

func ReadSaveFile(path string) (SaveFile, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return SaveFile{}, fmt.Errorf("read save: %w", err)
	}
	var sf SaveFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return SaveFile{}, fmt.Errorf("unmarshal save: %w", err)
	}
	return sf, nil
}

func ListSaveFiles(configured string) ([]SaveFileInfo, error) {
	dir, err := ResolveSaveDir(configured)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read save dir: %w", err)
	}

	var saves []SaveFileInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		saves = append(saves, SaveFileInfo{
			Name:    e.Name(),
			Path:    filepath.Join(dir, e.Name()),
			ModTime: info.ModTime(),
		})
	}

	sort.Slice(saves, func(i, j int) bool { return saves[i].ModTime.After(saves[j].ModTime) })
	return saves, nil
}
