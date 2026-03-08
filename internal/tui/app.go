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

type screen int

const (
	screenMenu screen = iota
	screenGame
	screenSettings
	screenAbout
)

type stateMsg game.StateSnapshot
type tickMsg time.Time
type errorMsg string

type Model struct {
	screen      screen
	menuIdx     int
	settingsIdx int
	cfg         game.Config

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

func NewModel(ctx context.Context, cfg game.Config) *Model {
	return &Model{
		screen: screenMenu,
		cfg:    cfg,
		ctx:    ctx,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func doTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenGame:
		return m.updateGame(msg)
	case screenSettings:
		return m.updateSettings(msg)
	case screenAbout:
		return m.updateAbout(msg)
	}

	return m, nil
}

func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.menuIdx > 0 {
				m.menuIdx--
			}
		case "down", "j":
			if m.menuIdx < 2 {
				m.menuIdx++
			}
		case "enter":
			switch m.menuIdx {
			case 0:
				return m.startGame()
			case 1:
				m.screen = screenSettings
			case 2:
				m.screen = screenAbout
			}
		}
	}
	return m, nil
}

func (m Model) updateGame(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

func (m Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.screen = screenMenu
		case "up", "k":
			if m.settingsIdx > 0 {
				m.settingsIdx--
			}
		case "down", "j":
			if m.settingsIdx < len(settingsItems)-1 {
				m.settingsIdx++
			}
		case "left", "h":
			settingsItems[m.settingsIdx].Dec(&m.cfg)
		case "right", "l":
			settingsItems[m.settingsIdx].Inc(&m.cfg)
		}
	}
	return m, nil
}

func (m Model) updateAbout(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.screen = screenMenu
		}
	}
	return m, nil
}

func (m Model) startGame() (tea.Model, tea.Cmd) {
	gameState := game.InitGame(m.cfg)
	gameState.Paused = true

	metrics := &middleware.Metrics{}

	engineActor, err := actor.StartNew[*game.State](
		m.ctx,
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
		m.statusMsg = fmt.Sprintf("Failed to start game: %v", err)
		return m, nil
	}

	snap := gameState.Snapshot()

	nodeIDs := make([]uint64, 0, len(snap.Nodes))
	for id := range snap.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })

	m.state = snap
	m.engineRef = engineActor
	m.tickRate = gameState.Config.TickRate
	m.nodeIDs = nodeIDs
	m.metrics = metrics
	m.screen = screenGame
	m.statusMsg = ""

	return m, doTick(m.tickRate)
}

func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Loading CYBERSPACE...")
	}

	var content string
	switch m.screen {
	case screenMenu:
		content = renderMenu(m.menuIdx, m.width, m.height)
	case screenGame:
		content = m.renderGame()
	case screenSettings:
		content = renderSettings(m.cfg, m.settingsIdx, m.width, m.height)
	case screenAbout:
		content = renderAbout(m.width, m.height)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderGame() string {
	if m.width < 60 || m.height < 20 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			styleError.Render("Terminal too small. Need at least 60x20."))
	}

	sidebarWidth := min(28, m.width/3)
	mainWidth := m.width - sidebarWidth - 2

	// Panel borders + padding consume 4 cols (border 2 + padding 2) and 2 rows (border top+bottom)
	innerWidth := mainWidth - 4
	panelHeight := m.height - 2 // leave room for status bar
	innerHeight := panelHeight - 2

	// HUD
	hud := renderHUD(m.state, innerWidth)

	// Vertical budget: HUD(1) + graph + details(3) + eventlog(eventHeight)
	detailHeight := 3
	eventHeight := 6
	graphHeight := innerHeight - 1 - detailHeight - eventHeight
	if graphHeight < 8 {
		graphHeight = 8
	}
	graph := renderGraph(m.state, m.selectedIdx, m.nodeIDs, innerWidth, graphHeight)

	// Selected node details
	details := renderSelectedDetails(m.state, m.selectedIdx, m.nodeIDs)

	// Event log
	eventLog := renderEventLog(m.state.Events, eventHeight)

	// Sidebar (guide) — constrain to panel inner height
	sidebar := renderSidebar(m.state, sidebarWidth-4)

	// Compose left panel with explicit height to match terminal
	leftPanel := stylePanel.Width(mainWidth).Height(innerHeight).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			hud,
			graph,
			details,
			eventLog,
		),
	)

	// Compose right panel with matching height
	rightPanel := stylePanel.Width(sidebarWidth).Height(innerHeight).Render(sidebar)

	// Join horizontally
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Status bar
	statusBar := ""
	if m.statusMsg != "" {
		statusBar = styleError.Render(m.statusMsg)
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
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

func (m *Model) Shutdown() {
	if m.engineRef != nil {
		_ = m.engineRef.Stop(5 * time.Second)
	}
}
