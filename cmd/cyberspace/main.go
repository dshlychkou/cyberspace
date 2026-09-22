package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"

	"github.com/barnowlsnest/go-configlib/v2/pkg/configs"

	"github.com/dshlychkou/cyberspace/v2/internal/game"
	"github.com/dshlychkou/cyberspace/v2/internal/tui"
)

func main() {
	cfg := &game.Config{}
	if _, err := configs.Resolve(cfg, "cyberspace"); err != nil {
		log.Fatalf("Config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	model := tui.NewModel(ctx, cfg)
	p := tea.NewProgram(model)

	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	_, err := p.Run()
	stop()
	model.Shutdown()
	if err != nil {
		log.Fatalf("Error running game: %v", err)
	}
}
