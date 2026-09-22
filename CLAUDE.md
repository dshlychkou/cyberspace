# CYBERSPACE

Terminal network strategy game. Infiltrate a cyberpunk network, deploy programs, hack through ICE defenses, spread viruses, and capture the CORE to win.

## Module
github.com/dshlychkou/cyberspace/v2

## Dependencies
- github.com/barnowlsnest/go-actorlib/v5 -- Actor model (game engine)
- github.com/barnowlsnest/go-datalib/v5 -- Data structures (DAG, Heap, BTree, Fenwick)
- charm.land/bubbletea/v2 -- TUI framework
- charm.land/lipgloss/v2 -- TUI styling (neon purple/cyberpunk theme)
- github.com/barnowlsnest/go-configlib/v2 -- CLI/env config
- github.com/barnowlsnest/go-logslib/v2 -- Structured logging (event log file)

## Commands
task run    # Build and run the game
task test   # Run tests
task lint   # Run linter
task sanity # ALWAYS run this after making changes

## Architecture
- **Actor model**: GoActor[*game.State] processes TickCmd, TogglePauseCmd, SpawnProgramCmd, DeployVirusCmd, SaveCmd, ShutdownCmd
- **Engine lifecycle**: actor runs on its own cancelable context (not the TUI SIGINT ctx). `stopEngine` delivers ShutdownCmd via `context.Background()`, then `CloseEventLog` as a safety net
- **TUI screens**: screenMenu → screenGame / screenSettings / screenAbout / screenLoad (dispatch in app.go)
- **Game engine deferred**: actor created only when Play/Load is selected (`startGame` / `startEngineWithState`)
- **Save/Load**: ESC pauses game (keeps actor alive), menu shows Continue/Save when game in progress. Saves to JSON in configurable `SaveDir` (default `~/.cyberspace/saves`). Load screen lists saves by date, supports delete. `ToSaveFile` deep-copies entities; `FromSaveFile` validates nil/unknown nodes
- **Canvas renderer**: 2D cell grid with Unicode braille edges (`canvas.go`), flat radial layout (`graph_view.go`), optional yaw rotate (`camera.go`)
- **HUD**: 2-line bar — tick/counts + btop-style CORE/Threat gauges + Data/Compute braille sparklines (`hud.go`, `sparkline.go`)
- **Workstation panel**: right column is `SYS://NODE-OPS` CRT console (LOCK/PROBE/PULSE + scrolling ops) — `workstation.go`; `sidebar.go` only retains shared helpers like `countEntities`
- **Economy**: Data (vaults +5/prog/tick), Compute (relays +3/prog/tick), upkeep (-1 Data/prog), core hold (-2 Compute/prog)
- **Event log file**: JSON event log via go-logslib, configured by `EventLogFile` (default `./cyberspace.log`)
- **Node selection**: `selectedNodeID uint64` (survives node destruction), spatial arrow nav + mouse click-to-select; selected node `[…]`, neighbors `(…)`, selected links solid cyan braille
- **Node positions cached in Update**: `nodePositions []nodePos` + `graphOffset` computed on stateMsg/WindowSizeMsg, passed to renderGraph

## Key Files
- `cmd/cyberspace/main.go` - entry point; signal ctx quits TUI; `model.Shutdown()` after `p.Run()`
- `internal/game/config.go` - all game settings with defaults (all fields have json tags)
- `internal/game/engine.go` - tick logic, ICE spawning, economy, win/lose, SaveCmd, ShutdownCmd
- `internal/game/state.go` - StateSnapshot with economy fields, event log I/O
- `internal/game/savefile.go` - SaveFile struct, ToSaveFile/FromSaveFile, file I/O
- `internal/tui/app.go` - Model, screen dispatch, Update/View, panel layout, engine start/stop, save/load
- `internal/tui/graph_view.go` - flat radial layout, braille edges, legend, selection chrome
- `internal/tui/camera.go` - yaw-only in-plane map rotation (h/l, 0 reset)
- `internal/tui/canvas.go` - 2D cell grid + braille `drawEdge` / `setBrailleDot`
- `internal/tui/hud.go` - 2-line HUD (gauges)
- `internal/tui/sparkline.go` - rolling Data/Compute braille sparklines
- `internal/tui/workstation.go` - CRT right-panel console
- `internal/tui/sidebar.go` - shared entity-count helpers used by graph/workstation
- `internal/tui/menu.go` - dynamic menu, settings, about screen
- `internal/tui/load_screen.go` - load game screen (list saves, select, delete)
- `internal/tui/styles.go` - neon color palette and lipgloss styles
- `internal/tui/eventlog.go` - scrolling event log (left panel)

## Conway Rules
- **Survival**: programs need 1–10 support (local programs − 1 + neighbor programs). Outside range → death
- **Auto-spread**: empty server/relay/vault with 3–10 neighbor programs gets a new program (checks both SpreadExact and SurviveMax to prevent born-dead spawns)
- **ICE suppress**: ICE ≥ programs on same node → all programs die
- **ICE patrol**: ICE moves 1 hop/tick toward undefended neighbors with programs
- **Virus corrupt**: virus flips 1 adjacent ICE → program per tick

## Balance Notes
- SurviveMax must accommodate network connectivity (hub nodes have 5–7 neighbors); too low causes overcrowding cascade
- Auto-spread checks `neighborPrograms <= SurviveMax` to prevent spawning programs that immediately die from overcrowding
- Death messages use pre-death snapshot counts for accuracy (not mid-removal counts)
- `ICESpawnTick == 0` is treated as disabled (no divide-by-zero); spawn/deploy blocked after GameOver
- `TestInitGameSurvives20Ticks` validates that ≥75% of random games survive 20 ticks without player input

## lipgloss v2 Gotcha
`lipgloss.Color()` is a FUNCTION returning `color.Color` (from image/color), NOT a type. Use `color.Color` for type annotations.
