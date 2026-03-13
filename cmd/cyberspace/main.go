package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"

	"github.com/barnowlsnest/go-configlib/v2/pkg/configs"

	"github.com/dshlychkou/cyberspace/internal/game"
	"github.com/dshlychkou/cyberspace/internal/tui"
)

func main() {
	cfg := &game.Config{}
	if _, err := configs.Resolve(cfg, "cyberspace"); err != nil {
		log.Fatalf("Config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	model := tui.NewModel(ctx, cfg)
	p := tea.NewProgram(model)

	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		<-ctx.Done()
		stop()
	}()
	go func() {
		<-doneChan
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running game: %v", err)
	}
}
