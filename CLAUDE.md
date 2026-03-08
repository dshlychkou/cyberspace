# CYBERSPACE

Terminal network strategy game. Infiltrate a cyberpunk network, deploy programs, hack through ICE defenses, spread viruses, and capture the CORE to win.

## Module
github.com/dshlychkou/cyberspace

## Dependencies
- github.com/barnowlsnest/go-actorlib/v4 -- Actor model (game engine)
- github.com/barnowlsnest/go-datalib -- Data structures (DAG, Heap, BTree, Fenwick)
- charm.land/bubbletea/v2 -- TUI framework
- charm.land/lipgloss/v2 -- TUI styling (neon purple/cyberpunk theme)
- github.com/barnowlsnest/go-configlib/v2 -- CLI/env config

## Commands
task run    # Build and run the game
task test   # Run tests
task lint   # Run linter
task sanity # ALWAYS run this after making changes

## Architecture
- **Actor model**: GoActor[*game.State] processes TickCmd, TogglePauseCmd, SpawnProgramCmd, DeployVirusCmd
- **TUI screens**: screenMenu → screenGame / screenSettings / screenAbout (dispatch in app.go)
- **Game engine deferred**: actor created only when Play is selected (startGame method)
- **Canvas renderer**: 2D cell grid in canvas.go, radial graph layout in graph_view.go
- **Economy**: Data (vaults +5/prog/tick), Compute (relays +2/prog/tick), upkeep (-1 Data/prog), core hold (-3 Compute/prog)

## Key Files
- `cmd/cyberspace/main.go` - entry point, passes Config to tui.NewModel
- `internal/game/config.go` - all game settings with defaults
- `internal/game/engine.go` - tick logic, ICE spawning, economy, win/lose
- `internal/game/state.go` - StateSnapshot with economy fields
- `internal/tui/app.go` - Model, screen dispatch, Update/View, panel layout
- `internal/tui/graph_view.go` - radial layout, canvas drawing, legend, flow animations
- `internal/tui/menu.go` - main menu, settings (interactive), about screen
- `internal/tui/sidebar.go` - right panel guide (nodes, entities, rules, economy)
- `internal/tui/hud.go` - top bar (tick, counts, resources, threat bar)
- `internal/tui/canvas.go` - 2D cell grid with line drawing
- `internal/tui/styles.go` - neon color palette and lipgloss styles
- `internal/tui/eventlog.go` - scrolling event log

## lipgloss v2 Gotcha
`lipgloss.Color()` is a FUNCTION returning `color.Color` (from image/color), NOT a type. Use `color.Color` for type annotations.
