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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := &game.Config{}
	if _, err := configs.Resolve(cfg, "cyberspace"); err != nil {
		exitErr("Config error: %v", err)
	}

	model := tui.NewModel(ctx, *cfg)

	p := tea.NewProgram(model)

	go func() {
		<-ctx.Done()
		p.Send(tea.QuitMsg{})
	}()

	if _, err := p.Run(); err != nil {
		exitErr("Error running game: %v", err)
	}

	model.Shutdown()
}

func exitErr(format string, args ...any) {
	log.Fatalf(format, args...)
}
