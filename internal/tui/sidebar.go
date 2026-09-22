package tui

import (
	"github.com/dshlychkou/cyberspace/v2/internal/game"
)

func countEntities(n game.NodeSnapshot, snap *game.StateSnapshot) (programs, ices, viruses int) {
	programByID := make(map[int]bool, len(snap.Programs))
	for _, p := range snap.Programs {
		programByID[p.ID] = true
	}
	iceByID := make(map[int]bool, len(snap.ICEs))
	for _, ice := range snap.ICEs {
		iceByID[ice.ID] = true
	}
	virusByID := make(map[int]bool, len(snap.Viruses))
	for _, v := range snap.Viruses {
		virusByID[v.ID] = true
	}
	for _, eid := range n.Entities {
		switch {
		case programByID[eid]:
			programs++
		case iceByID[eid]:
			ices++
		case virusByID[eid]:
			viruses++
		}
	}
	return
}
