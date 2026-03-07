package tui

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/barnowlsnest/go-actorlib/v4/pkg/actor"
	"github.com/barnowlsnest/go-actorlib/v4/pkg/middleware"
	"github.com/dshlychkou/cyberspace/internal/game"
)

type stateMsg game.StateSnapshot
type tickMsg time.Time
type errorMsg string

type Model struct {
	state       game.StateSnapshot
	engineRef   *actor.GoActor[*game.State]
	ctx         context.Context
	width       int
	height      int
	selectedIdx int
	nodeIDs     []uint64
	tickRate    time.Duration
	statusMsg   string
	metrics     *middleware.Metrics
}

type StateProvider struct {
	State *game.State
}

func (p *StateProvider) Provide() *game.State { return p.State }

func NewModel(gameState *game.State) (*Model, error) {
	ctx := context.Background()
	metrics := &middleware.Metrics{}

	// Start paused so the player can read the guide
	gameState.Paused = true

	engineActor, err := actor.StartNew[*game.State](
		ctx,
		5*time.Second,
		actor.WithProvider[*game.State](&StateProvider{State: gameState}),
		actor.WithName[*game.State]("game-engine"),
		actor.WithInputBufferSize[*game.State](32),
		actor.WithReceiveTimeout[*game.State](10*time.Second),
		actor.WithMiddleware(
			middleware.Recovery[*game.State](slog.Default()),
			middleware.MetricsMiddleware[*game.State](metrics),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start game engine actor: %w", err)
	}

	snap := gameState.Snapshot()

	nodeIDs := make([]uint64, 0, len(snap.Nodes))
	for id := range snap.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })

	return &Model{
		state:     snap,
		engineRef: engineActor,
		ctx:       ctx,
		tickRate:  gameState.Config.TickRate,
		nodeIDs:   nodeIDs,
		metrics:   metrics,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return doTick(m.tickRate)
}

func doTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			m.sendTick(),
			doTick(m.tickRate),
		)

	case stateMsg:
		m.state = game.StateSnapshot(msg)
		nodeIDs := make([]uint64, 0, len(m.state.Nodes))
		for id := range m.state.Nodes {
			nodeIDs = append(nodeIDs, id)
		}
		sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })
		m.nodeIDs = nodeIDs
		m.statusMsg = ""
		return m, nil

	case errorMsg:
		m.statusMsg = string(msg)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.engineRef != nil {
				_ = m.engineRef.Stop(5 * time.Second)
			}
			return m, tea.Quit

		case "space":
			return m, m.sendTogglePause()

		case "s":
			if m.selectedIdx < len(m.nodeIDs) {
				return m, m.sendSpawnProgram(m.nodeIDs[m.selectedIdx])
			}

		case "v":
			if m.selectedIdx < len(m.nodeIDs) {
				return m, m.sendDeployVirus(m.nodeIDs[m.selectedIdx])
			}

		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}

		case "down", "j":
			if m.selectedIdx < len(m.nodeIDs)-1 {
				m.selectedIdx++
			}

		case "+", "=":
			if m.tickRate > 100*time.Millisecond {
				m.tickRate -= 100 * time.Millisecond
			}

		case "-":
			if m.tickRate < 2*time.Second {
				m.tickRate += 100 * time.Millisecond
			}
		}
	}

	return m, nil
}

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Loading CYBERSPACE...")
	}

	sidebarWidth := min(28, m.width/3)
	mainWidth := m.width - sidebarWidth - 4

	// HUD
	hud := renderHUD(m.state, mainWidth-2)

	// Node list
	nodeList := renderNodeList(m.state, m.selectedIdx, m.nodeIDs, mainWidth-2)

	// Selected node details
	details := renderSelectedDetails(m.state, m.selectedIdx, m.nodeIDs)

	// Event log
	eventLog := renderEventLog(m.state.Events, 10)

	// Sidebar (guide)
	sidebar := renderSidebar(m.state, sidebarWidth-2)

	// Compose left panel
	leftPanel := stylePanel.Width(mainWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			hud,
			"",
			nodeList,
			details,
			eventLog,
		),
	)

	// Compose right panel
	rightPanel := stylePanel.Width(sidebarWidth).Render(sidebar)

	// Join horizontally
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Status bar
	statusBar := ""
	if m.statusMsg != "" {
		statusBar = styleError.Render(m.statusMsg)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, body, statusBar)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) sendTick() tea.Cmd {
	return func() tea.Msg {
		done := make(chan game.StateSnapshot, 1)
		cmd := &game.TickCmd{
			OnComplete: func(snap game.StateSnapshot) {
				done <- snap
			},
		}
		if err := m.engineRef.Receive(m.ctx, cmd); err != nil {
			return errorMsg(fmt.Sprintf("tick error: %v", err))
		}
		select {
		case snap := <-done:
			return stateMsg(snap)
		case <-time.After(5 * time.Second):
			return errorMsg("tick timeout")
		}
	}
}

func (m Model) sendTogglePause() tea.Cmd {
	return func() tea.Msg {
		cmd := &game.TogglePauseCmd{}
		if err := m.engineRef.Receive(m.ctx, cmd); err != nil {
			return errorMsg(fmt.Sprintf("pause error: %v", err))
		}
		return nil
	}
}

func (m Model) sendSpawnProgram(nodeID uint64) tea.Cmd {
	return func() tea.Msg {
		done := make(chan string, 1)
		cmd := &game.SpawnProgramCmd{
			NodeID: nodeID,
			OnComplete: func(ok bool, msg string) {
				if !ok {
					done <- msg
				} else {
					done <- ""
				}
			},
		}
		if err := m.engineRef.Receive(m.ctx, cmd); err != nil {
			return errorMsg(fmt.Sprintf("spawn error: %v", err))
		}
		select {
		case msg := <-done:
			if msg != "" {
				return errorMsg(msg)
			}
			return nil
		case <-time.After(5 * time.Second):
			return errorMsg("spawn timeout")
		}
	}
}

func (m Model) sendDeployVirus(nodeID uint64) tea.Cmd {
	return func() tea.Msg {
		done := make(chan string, 1)
		cmd := &game.DeployVirusCmd{
			NodeID: nodeID,
			OnComplete: func(ok bool, msg string) {
				if !ok {
					done <- msg
				} else {
					done <- ""
				}
			},
		}
		if err := m.engineRef.Receive(m.ctx, cmd); err != nil {
			return errorMsg(fmt.Sprintf("virus error: %v", err))
		}
		select {
		case msg := <-done:
			if msg != "" {
				return errorMsg(msg)
			}
			return nil
		case <-time.After(5 * time.Second):
			return errorMsg("deploy timeout")
		}
	}
}
