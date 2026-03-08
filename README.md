# cyberspace

Cyberpunk graph automaton game. Conway's Game of Life on a network graph, in your terminal.

## Run

```bash
task run
# or
go run ./cmd/cyberspace
```

## How to play

You control **programs** spreading through a network of nodes. Your goal is to get programs to the **CORE** node and hold it.

- Programs auto-spread between servers, relays, and vaults
- **Firewalls** and **CORE** block auto-spread — you must place programs there manually with `S`
- **ICE** (enemy defenses) destroys your programs when it outnumbers them on a node
- **Viruses** convert nearby ICE into programs

### Controls

| Key     | Action                  |
|---------|-------------------------|
| `↑`/`↓` | Select node             |
| `S`     | Spawn program (costs Data) |
| `V`     | Deploy virus (costs Compute) |
| `Space` | Pause / Resume          |
| `+`/`-` | Speed up / slow down    |
| `Q`     | Quit                    |

### Resources

- **Data** — earned by programs on vault nodes (+5/tick). Spent to spawn programs.
- **Compute** — earned by programs on relay nodes (+2/tick). Spent to deploy viruses.

## Configuration

All settings are configurable via CLI flags or environment variables, powered by [go-configlib/v2](https://github.com/barnowlsnest/go-configlib).

### CLI flags

```bash
go run ./cmd/cyberspace --cyberspace_tick_rate=500ms --cyberspace_initial_programs=5
```

### Environment variables

```bash
CYBERSPACE_TICK_RATE=2s CYBERSPACE_INITIAL_ICE=4 go run ./cmd/cyberspace
```

### All settings

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--cyberspace_tick_rate` | `CYBERSPACE_TICK_RATE` | `1s` | Tick interval (e.g. 500ms, 1s, 2s) |
| `--cyberspace_initial_programs` | `CYBERSPACE_INITIAL_PROGRAMS` | `3` | Starting program count |
| `--cyberspace_initial_ice` | `CYBERSPACE_INITIAL_ICE` | `2` | Starting ICE count |
| `--cyberspace_virus_lifespan` | `CYBERSPACE_VIRUS_LIFESPAN` | `8` | Ticks before a virus decays |
| `--cyberspace_core_win_threshold` | `CYBERSPACE_CORE_WIN_THRESHOLD` | `2` | Programs needed on core to start winning |
| `--cyberspace_core_win_duration` | `CYBERSPACE_CORE_WIN_DURATION` | `10` | Consecutive ticks holding core to win |
| `--cyberspace_data_harvest_rate` | `CYBERSPACE_DATA_HARVEST_RATE` | `5` | Data earned per tick per program on a vault |
| `--cyberspace_program_spawn_cost` | `CYBERSPACE_PROGRAM_SPAWN_COST` | `15` | Data cost to spawn a program |
| `--cyberspace_virus_deploy_cost` | `CYBERSPACE_VIRUS_DEPLOY_COST` | `25` | Compute cost to deploy a virus |
| `--cyberspace_survive_min` | `CYBERSPACE_SURVIVE_MIN` | `1` | Min neighbor support for program survival |
| `--cyberspace_survive_max` | `CYBERSPACE_SURVIVE_MAX` | `6` | Max neighbor support before overcrowding |
| `--cyberspace_spread_exact` | `CYBERSPACE_SPREAD_EXACT` | `2` | Min neighbor programs for auto-spread |
| `--cyberspace_initial_data` | `CYBERSPACE_INITIAL_DATA` | `100` | Starting data resource |
| `--cyberspace_initial_compute` | `CYBERSPACE_INITIAL_COMPUTE` | `50` | Starting compute resource |
| `--cyberspace_ice_spawn_tick` | `CYBERSPACE_ICE_SPAWN_TICK` | `30` | Tick when first new ICE spawns |
| `--cyberspace_ice_escalation_tick` | `CYBERSPACE_ICE_ESCALATION_TICK` | `80` | Tick when ICE spawns start accelerating |

## Test

```bash
task test
# or
go test ./...
```
