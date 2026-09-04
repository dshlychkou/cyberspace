package tui

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
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
	screenLoad
)

const (
	keyCtrlC = "ctrl+c"
	keyUp    = "up"
	keyDown  = "down"
	keyLeft  = "left"
	keyRight = "right"
	keyEsc   = "esc"
	keyEnter = "enter"
)

type stateMsg game.StateSnapshot
type tickMsg time.Time
type errorMsg string
type saveSuccessMsg string

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
	saveFiles      []game.SaveFileInfo
	loadIdx        int
}

func sortedNodeIDs[T any](nodes map[uint64]T) []uint64 {
	ids := make([]uint64, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
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

func (m *Model) gameInProgress() bool {
	return m.engineRef != nil
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
	case screenLoad:
		return m.updateLoad(msg)
	}

	return m, nil
}

func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	items := m.menuItems()

	switch msg := msg.(type) {
	case saveSuccessMsg:
		m.statusMsg = string(msg)
		return m, nil

	case errorMsg:
		m.statusMsg = string(msg)
		return m, nil

	case tea.KeyPressMsg:
		// Clamp index to valid range
		if m.menuIdx >= len(items) {
			m.menuIdx = len(items) - 1
		}

		switch msg.String() {
		case "q", keyCtrlC:
			m.destroyGame()
			return m, tea.Quit
		case keyUp, "k":
			if m.menuIdx > 0 {
				m.menuIdx--
			}
		case keyDown, "j":
			if m.menuIdx < len(items)-1 {
				m.menuIdx++
			}
		case keyEnter:
			return m.handleMenuAction(items[m.menuIdx].Action)
		}
	}
	return m, nil
}

func (m *Model) handleMenuAction(action menuAction) (tea.Model, tea.Cmd) {
	switch action {
	case menuContinue:
		return m.continueGame()
	case menuSave:
		cmd := m.saveGame()
		return m, cmd
	case menuNewGame:
		m.destroyGame()
		return m.startGame()
	case menuLoad:
		m.screen = screenLoad
		m.loadSaves()
	case menuSettings:
		m.screen = screenSettings
	case menuAbout:
		m.screen = screenAbout
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
		nodeIDs := sortedNodeIDs(m.state.Nodes)
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
		return m.pauseAndReturnToMenu()
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

func (m *Model) pauseAndReturnToMenu() (tea.Model, tea.Cmd) {
	// Pause the game if it's not already paused
	var cmd tea.Cmd
	if !m.state.Paused {
		cmd = m.sendTogglePause()
	}
	m.screen = screenMenu
	m.menuIdx = 0
	m.statusMsg = ""
	return m, cmd
}

func (m *Model) destroyGame() {
	m.stopEngine()
	m.state = game.StateSnapshot{}
	m.selectedNodeID = 0
	m.nodePositions = nil
	m.nodeIDs = nil
	m.statusMsg = ""
}

func (m *Model) continueGame() (tea.Model, tea.Cmd) {
	m.screen = screenGame
	m.statusMsg = ""
	return m, doTick(m.tickRate)
}

func (m *Model) handleRestart() (tea.Model, tea.Cmd) {
	if m.state.GameOver {
		m.destroyGame()
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
	gameState, err := game.InitGame(&m.cfg)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Failed to init game: %v", err)
		return m, nil
	}
	gameState.Paused = true

	return m.startEngineWithState(gameState)
}

func (m *Model) startEngineWithState(gameState *game.State) (tea.Model, tea.Cmd) {
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

	nodeIDs := sortedNodeIDs(snap.Nodes)

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

func (m *Model) saveGame() tea.Cmd {
	return func() tea.Msg {
		done := make(chan game.SaveFile, 1)
		cmd := &game.SaveCmd{
			OnComplete: func(sf game.SaveFile) {
				done <- sf
			},
		}
		if err := m.engineRef.Receive(m.ctx, cmd); err != nil {
			return errorMsg(fmt.Sprintf("save error: %v", err))
		}

		var sf game.SaveFile
		select {
		case sf = <-done:
		case <-time.After(5 * time.Second):
			return errorMsg("save timeout")
		}

		dir, err := game.ResolveSaveDir(m.cfg.SaveDir)
		if err != nil {
			return errorMsg(fmt.Sprintf("save dir error: %v", err))
		}

		filename := time.Now().Format("2006-01-02T15-04-05") + ".json"
		path := dir + "/" + filename
		if err := game.WriteSaveFile(path, &sf); err != nil {
			return errorMsg(fmt.Sprintf("save write error: %v", err))
		}

		return saveSuccessMsg(fmt.Sprintf("Game saved: %s", filename))
	}
}

func (m *Model) loadGame(path string) (tea.Model, tea.Cmd) {
	m.destroyGame()

	sf, err := game.ReadSaveFile(path)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Failed to load: %v", err)
		m.screen = screenMenu
		return m, nil
	}

	gameState, err2 := game.FromSaveFile(&sf)
	if err2 != nil {
		m.statusMsg = fmt.Sprintf("Failed to restore game: %v", err2)
		m.screen = screenMenu
		return m, nil
	}
	gameState.Paused = true

	return m.startEngineWithState(gameState)
}

func (m *Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Loading CYBERSPACE...")
	}

	var content string
	switch m.screen {
	case screenMenu:
		content = renderMenu(m.menuItems(), m.menuIdx, m.width, m.height, m.statusMsg)
	case screenGame:
		content = m.renderGame()
	case screenSettings:
		content = renderSettings(&m.cfg, m.settingsIdx, m.width, m.height)
	case screenAbout:
		content = renderAbout(m.width, m.height)
	case screenLoad:
		content = renderLoadScreen(m.saveFiles, m.loadIdx, m.width, m.height)
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

	d := m.panelDimensions()

	// HUD
	hud := renderHUD(&m.state, d.innerWidth)

	graph := renderGraph(&m.state, m.selectedNodeID, m.nodePositions, d.innerWidth, d.graphHeight)

	// Selected node details
	details := renderSelectedDetails(&m.state, m.selectedNodeID)

	// Event log
	eventLog := renderEventLog(m.state.Events, d.eventHeight)

	// Sidebar (guide) — constrain to panel inner height
	sidebar := renderSidebar(&m.state, d.sidebarWidth-4)

	// Compose left panel with explicit height to match terminal
	leftPanel := stylePanel.Width(d.mainWidth).Height(d.innerHeight).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			hud,
			graph,
			details,
			eventLog,
		),
	)

	// Compose right panel with matching height
	rightPanel := stylePanel.Width(d.sidebarWidth).Height(d.innerHeight).Render(sidebar)

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

// sendCmd builds an entity command via build (wiring its OnComplete to an
// internal channel) and delivers it to the engine, translating the result
// into a tea.Msg.
func (m *Model) sendCmd(build func(onComplete func(bool, string)) actor.Executable[*game.State], errPrefix string) tea.Cmd {
	return func() tea.Msg {
		done := make(chan string, 1)
		cmd := build(func(ok bool, msg string) {
			if !ok {
				done <- msg
			} else {
				done <- ""
			}
		})
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
	return m.sendCmd(func(oc func(bool, string)) actor.Executable[*game.State] {
		return &game.SpawnProgramCmd{NodeID: nodeID, OnComplete: oc}
	}, "spawn")
}

func (m *Model) sendDeployVirus(nodeID uint64) tea.Cmd {
	return m.sendCmd(func(oc func(bool, string)) actor.Executable[*game.State] {
		return &game.DeployVirusCmd{NodeID: nodeID, OnComplete: oc}
	}, "deploy")
}

// panelDims holds the panel/graph layout measurements shared by renderGame
// (the View path) and hitTestNode/computeNodePositions (the input path), so
// the two stay in sync.
type panelDims struct {
	sidebarWidth int
	mainWidth    int
	innerWidth   int
	innerHeight  int
	graphHeight  int
	eventHeight  int
}

func (m *Model) panelDimensions() panelDims {
	sidebarWidth := min(28, m.width/3)
	mainWidth := m.width - sidebarWidth - 2

	// Panel borders + padding consume 4 cols (border 2 + padding 2) and 2 rows (border top+bottom)
	innerWidth := mainWidth - 4
	panelHeight := m.height - 2 // leave room for status bar
	innerHeight := panelHeight - 2

	// Vertical budget: HUD(1) + graph + details(3) + eventlog(eventHeight)
	const detailHeight = 3
	const eventHeight = 6
	graphHeight := max(innerHeight-1-detailHeight-eventHeight, 8)

	return panelDims{
		sidebarWidth: sidebarWidth,
		mainWidth:    mainWidth,
		innerWidth:   innerWidth,
		innerHeight:  innerHeight,
		graphHeight:  graphHeight,
		eventHeight:  eventHeight,
	}
}

func (m *Model) graphDimensions() (graphWidth, graphHeight int) {
	d := m.panelDimensions()
	return d.innerWidth, d.graphHeight
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
