package game

import "time"

type Config struct {
	TickRate          time.Duration `name:"tick_rate"          default:"1s"   usage:"tick interval (e.g. 500ms, 1s, 2s)"`
	InitialPrograms   int           `name:"initial_programs"   default:"3"    usage:"starting program count"`
	InitialICE        int           `name:"initial_ice"        default:"3"    usage:"starting ICE count"`
	VirusLifespan     int           `name:"virus_lifespan"     default:"8"    usage:"ticks before a virus decays"`
	CoreWinThreshold  int           `name:"core_win_threshold" default:"4"    usage:"programs needed on core to start winning"`
	CoreWinDuration   int           `name:"core_win_duration"  default:"20"   usage:"consecutive ticks holding core to win"`
	DataHarvestRate   int           `name:"data_harvest_rate"  default:"5"    usage:"data earned per tick per program on a vault"`
	ProgramSpawnCost  int           `name:"program_spawn_cost" default:"20"   usage:"data cost to spawn a program"`
	VirusDeployCost   int           `name:"virus_deploy_cost"  default:"25"   usage:"compute cost to deploy a virus"`
	ProgramUpkeep     int           `name:"program_upkeep"     default:"1"    usage:"data cost per program per tick"`
	CoreHoldCost      int           `name:"core_hold_cost"     default:"3"    usage:"compute cost per program on core per tick"`
	SurviveMin        int           `name:"survive_min"        default:"1"    usage:"min neighbor support for program survival"`
	SurviveMax        int           `name:"survive_max"        default:"6"    usage:"max neighbor support before overcrowding"`
	SpreadExact       int           `name:"spread_exact"       default:"3"    usage:"min neighbor programs for auto-spread"`
	InitialData       int           `name:"initial_data"       default:"50"   usage:"starting data resource"`
	InitialCompute    int           `name:"initial_compute"    default:"25"   usage:"starting compute resource"`
	ICESpawnTick      int           `name:"ice_spawn_tick"     default:"8"    usage:"tick when first new ICE spawns"`
	ICEEscalationTick int           `name:"ice_escalation_tick" default:"25"  usage:"tick when ICE spawns start accelerating"`
}
