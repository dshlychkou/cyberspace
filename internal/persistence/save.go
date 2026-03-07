package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/dshlychkou/cyberspace/internal/game"
)

const stateDir = ".cyberspace"

type SaveData struct {
	Tick      int            `json:"tick"`
	Score     int            `json:"score"`
	Resources game.Resources `json:"resources"`
}

func SaveState(state game.StateSnapshot) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}

	data := SaveData{
		Tick:      state.Tick,
		Score:     state.Score,
		Resources: state.Resources,
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(stateDir, "state.json"), b, 0o644)
}

func LoadState() (*SaveData, error) {
	b, err := os.ReadFile(filepath.Join(stateDir, "state.json"))
	if err != nil {
		return nil, err
	}

	var data SaveData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
