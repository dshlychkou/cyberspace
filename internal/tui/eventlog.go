package tui

import (
	"fmt"
	"strings"

	"github.com/dshlychkou/cyberspace/internal/game"
)

func renderEventLog(events []game.Event, height int) string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("EVENT LOG"))
	sb.WriteByte('\n')

	start := 0
	maxEvents := height - 2
	if maxEvents < 1 {
		maxEvents = 1
	}
	if len(events) > maxEvents {
		start = len(events) - maxEvents
	}

	for _, e := range events[start:] {
		line := styleEvent.Render(fmt.Sprintf("[%d] %s", e.Tick, e.Message))
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return sb.String()
}
