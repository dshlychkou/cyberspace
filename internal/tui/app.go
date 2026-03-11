package tui

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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

const (
	keyCtrlC = "ctrl+c"
	keyUp    = "up"
	keyDown  = "down"
	keyLeft  = "left"
	keyRight = "right"
	keyEsc   = "esc"
)

type stateMsg game.StateSnapshot
type tickMsg time.Time
type errorMsg string

type Model struct {
	screen      screen
	menuIdx     int
	settingsIdx int
	cfg         game.Config

	state          game.StateSnapshot
	engineRef      *actor.GoActor[*game.State]
	ctx            context.Context
	width          int
	height         int
	selectedNodeID uint64
	nodeIDs        []uint64
	nodePositions  []nodePos
	graphOffset    struct{ x, y int }
	tickRate       time.Duration
	statusMsg      string
	metrics        *middleware.Metrics
}

type StateProvider struct {
	State *game.State
}

func (p *StateProvider) Provide() *game.State { return p.State }

func NewModel(ctx context.Context, cfg *game.Config) *Model {
	return &Model{
		screen: screenMenu,
		cfg:    *cfg,
		ctx:    ctx,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func doTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		if m.screen == screenGame && len(m.state.Nodes) > 0 {
			m.computeNodePositions()
			m.computeGraphOffset()
		}
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

func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "q", keyCtrlC:
			return m, tea.Quit
		case keyUp, "k":
			if m.menuIdx > 0 {
				m.menuIdx--
			}
		case keyDown, "j":
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

func (m *Model) updateGame(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// Validate selectedNodeID still exists; fallback to first node
		if _, ok := m.state.Nodes[m.selectedNodeID]; !ok && len(nodeIDs) > 0 {
			m.selectedNodeID = nodeIDs[0]
		}
		m.computeNodePositions()
		m.computeGraphOffset()
		m.statusMsg = ""
		return m, nil

	case errorMsg:
		m.statusMsg = string(msg)
		return m, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			mouse := msg.Mouse()
			if id, ok := m.hitTestNode(mouse.X, mouse.Y); ok {
				m.selectedNodeID = id
			}
		}

	case tea.KeyPressMsg:
		return m.handleGameKey(msg)
	}

	return m, nil
}

var arrowDirs = map[string]struct{ dx, dy int }{
	keyUp:    {0, -1},
	keyDown:  {0, 1},
	keyLeft:  {-1, 0},
	keyRight: {1, 0},
}

func (m *Model) handleGameKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if dir, ok := arrowDirs[key]; ok {
		m.selectedNodeID = m.spatialSelect(dir.dx, dir.dy)
		return m, nil
	}

	switch key {
	case keyEsc:
		return m.exitToMenu()
	case "r":
		return m.handleRestart()
	case "space":
		cmd := m.sendTogglePause()
		return m, cmd
	case "s":
		if m.selectedNodeID != 0 {
			cmd := m.sendSpawnProgram(m.selectedNodeID)
			return m, cmd
		}
	case "v":
		if m.selectedNodeID != 0 {
			cmd := m.sendDeployVirus(m.selectedNodeID)
			return m, cmd
		}
	case "+", "=":
		m.adjustSpeed(-100 * time.Millisecond)
	case "-":
		m.adjustSpeed(100 * time.Millisecond)
	}
	return m, nil
}

func (m *Model) exitToMenu() (tea.Model, tea.Cmd) {
	m.stopEngine()
	m.screen = screenMenu
	m.state = game.StateSnapshot{}
	m.selectedNodeID = 0
	m.nodePositions = nil
	m.nodeIDs = nil
	m.statusMsg = ""
	return m, nil
}

func (m *Model) handleRestart() (tea.Model, tea.Cmd) {
	if m.state.GameOver {
		if m.engineRef != nil {
			_ = m.engineRef.Stop(5 * time.Second)
			m.engineRef = nil
		}
		return m.startGame()
	}
	return m, nil
}

func (m *Model) adjustSpeed(delta time.Duration) {
	m.tickRate += delta
	if m.tickRate < 100*time.Millisecond {
		m.tickRate = 100 * time.Millisecond
	}
	if m.tickRate > 2*time.Second {
		m.tickRate = 2 * time.Second
	}
}

func (m *Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case keyCtrlC:
			return m, tea.Quit
		case keyEsc:
			m.screen = screenMenu
		case keyUp, "k":
			if m.settingsIdx > 0 {
				m.settingsIdx--
			}
		case keyDown, "j":
			if m.settingsIdx < len(settingsItems)-1 {
				m.settingsIdx++
			}
		case keyLeft, "h":
			settingsItems[m.settingsIdx].Dec(&m.cfg)
		case keyRight, "l":
			settingsItems[m.settingsIdx].Inc(&m.cfg)
		}
	}
	return m, nil
}

func (m *Model) updateAbout(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case keyCtrlC:
			return m, tea.Quit
		case keyEsc:
			m.screen = screenMenu
		}
	}
	return m, nil
}

func (m *Model) startGame() (tea.Model, tea.Cmd) {
	gameState := game.InitGame(&m.cfg)
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
	if len(nodeIDs) > 0 {
		m.selectedNodeID = nodeIDs[0]
	}

	return m, doTick(m.tickRate)
}

func (m *Model) View() tea.View {
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
		content = renderSettings(&m.cfg, m.settingsIdx, m.width, m.height)
	case screenAbout:
		content = renderAbout(m.width, m.height)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	if m.screen == screenGame {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

func (m *Model) renderGame() string {
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
	hud := renderHUD(&m.state, innerWidth)

	// Vertical budget: HUD(1) + graph + details(3) + eventlog(eventHeight)
	detailHeight := 3
	eventHeight := 6
	graphHeight := innerHeight - 1 - detailHeight - eventHeight
	if graphHeight < 8 {
		graphHeight = 8
	}
	graph := renderGraph(&m.state, m.selectedNodeID, m.nodePositions, innerWidth, graphHeight)

	// Selected node details
	details := renderSelectedDetails(&m.state, m.selectedNodeID)

	// Event log
	eventLog := renderEventLog(m.state.Events, eventHeight)

	// Sidebar (guide) — constrain to panel inner height
	sidebar := renderSidebar(&m.state, sidebarWidth-4)

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

func (m *Model) sendTick() tea.Cmd {
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

func (m *Model) sendTogglePause() tea.Cmd {
	return func() tea.Msg {
		cmd := &game.TogglePauseCmd{}
		if err := m.engineRef.Receive(m.ctx, cmd); err != nil {
			return errorMsg(fmt.Sprintf("pause error: %v", err))
		}
		return nil
	}
}

type completionCmd interface {
	actor.Executable[*game.State]
}

func newCompletionCmd(nodeID uint64, done chan string, spawn bool) completionCmd {
	onComplete := func(ok bool, msg string) {
		if !ok {
			done <- msg
		} else {
			done <- ""
		}
	}
	if spawn {
		return &game.SpawnProgramCmd{NodeID: nodeID, OnComplete: onComplete}
	}
	return &game.DeployVirusCmd{NodeID: nodeID, OnComplete: onComplete}
}

func (m *Model) sendEntityCmd(nodeID uint64, spawn bool, errPrefix string) tea.Cmd {
	return func() tea.Msg {
		done := make(chan string, 1)
		cmd := newCompletionCmd(nodeID, done, spawn)
		if err := m.engineRef.Receive(m.ctx, cmd); err != nil {
			return errorMsg(fmt.Sprintf("%s error: %v", errPrefix, err))
		}
		select {
		case msg := <-done:
			if msg != "" {
				return errorMsg(msg)
			}
			return nil
		case <-time.After(5 * time.Second):
			return errorMsg(errPrefix + " timeout")
		}
	}
}

func (m *Model) sendSpawnProgram(nodeID uint64) tea.Cmd {
	return m.sendEntityCmd(nodeID, true, "spawn")
}

func (m *Model) sendDeployVirus(nodeID uint64) tea.Cmd {
	return m.sendEntityCmd(nodeID, false, "deploy")
}

func (m *Model) graphDimensions() (graphWidth, graphHeight int) {
	sidebarWidth := min(28, m.width/3)
	mainWidth := m.width - sidebarWidth - 2
	innerWidth := mainWidth - 4
	panelHeight := m.height - 2
	innerHeight := panelHeight - 2

	graphWidth = innerWidth
	detailHeight := 3
	eventHeight := 6
	graphHeight = innerHeight - 1 - detailHeight - eventHeight
	if graphHeight < 8 {
		graphHeight = 8
	}
	return
}

func (m *Model) computeNodePositions() {
	gw, gh := m.graphDimensions()
	m.nodePositions = layoutNodes(&m.state, gw, gh)
}

func (m *Model) computeGraphOffset() {
	// stylePanel: Border(RoundedBorder()) = 1 cell each side, Padding(0, 1) = 1 cell left/right
	// x offset: border(1) + padding(1) = 2
	// y offset: border(1) + HUD line(1) = 2
	m.graphOffset.x = 2
	m.graphOffset.y = 2
}

func (m *Model) hitTestNode(termX, termY int) (uint64, bool) {
	localX := termX - m.graphOffset.x
	localY := termY - m.graphOffset.y

	gw, gh := m.graphDimensions()
	if localX < 0 || localY < 0 || localX >= gw || localY >= gh {
		return 0, false
	}

	var bestID uint64
	bestDist := math.MaxFloat64
	const maxDist = 4.0

	for _, p := range m.nodePositions {
		dx := float64(localX-p.x) * 0.5 // weight horizontal by 0.5 for terminal aspect ratio
		dy := float64(localY - p.y)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < bestDist && dist <= maxDist {
			bestDist = dist
			bestID = p.id
		}
	}

	if bestDist <= maxDist {
		return bestID, true
	}
	return 0, false
}

func (m *Model) spatialSelect(dirX, dirY int) uint64 {
	if len(m.nodePositions) == 0 {
		return m.selectedNodeID
	}

	// Find current node position
	var cur nodePos
	found := false
	for _, p := range m.nodePositions {
		if p.id == m.selectedNodeID {
			cur = p
			found = true
			break
		}
	}
	if !found {
		return m.selectedNodeID
	}

	bestID := m.selectedNodeID
	bestScore := math.MaxFloat64

	for _, p := range m.nodePositions {
		if p.id == m.selectedNodeID {
			continue
		}

		dx := float64(p.x - cur.x)
		dy := float64(p.y - cur.y)

		// Dot product with direction vector — must be positive (same half-plane)
		dot := dx*float64(dirX) + dy*float64(dirY)
		if dot <= 0 {
			continue
		}

		dist := math.Sqrt(dx*dx + dy*dy)
		if dist == 0 {
			continue
		}

		// Cross product magnitude for angular penalty
		cross := math.Abs(dx*float64(dirY) - dy*float64(dirX))
		angularPenalty := cross / dist * 2.0

		score := dist + angularPenalty*dist
		if score < bestScore {
			bestScore = score
			bestID = p.id
		}
	}

	return bestID
}

func (m *Model) stopEngine() {
	if m.engineRef != nil {
		_ = m.engineRef.Receive(m.ctx, &game.ShutdownCmd{})
		_ = m.engineRef.Stop(5 * time.Second)
		m.engineRef = nil
	}
}

func (m *Model) Shutdown() {
	m.stopEngine()
}
