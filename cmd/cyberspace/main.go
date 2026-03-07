package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/dshlychkou/cyberspace/internal/game"
	"github.com/dshlychkou/cyberspace/internal/tui"
)

func main() {
	cfg := game.DefaultConfig()
	gameState := game.InitGame(cfg)

	model, err := tui.NewModel(gameState)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running game: %v\n", err)
		os.Exit(1)
	}
}
