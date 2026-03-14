package game

import "time"

//nolint:lll // struct tags are inherently long
type Config struct {
	TickRate            time.Duration `name:"tick_rate" default:"1s" usage:"tick interval (e.g. 500ms, 1s, 2s)" json:"tick_rate"`
	InitialPrograms     int           `name:"initial_programs" default:"5" usage:"starting program count" json:"initial_programs"`
	InitialICE          int           `name:"initial_ice" default:"3" usage:"starting ICE count" json:"initial_ice"`
	VirusLifespan       int           `name:"virus_lifespan" default:"8" usage:"ticks before a virus decays" json:"virus_lifespan"`
	CoreWinThreshold    int           `name:"core_win_threshold" default:"3" usage:"programs needed on core to start winning" json:"core_win_threshold"`
	CoreWinDuration     int           `name:"core_win_duration" default:"8" usage:"consecutive ticks holding core to win" json:"core_win_duration"`
	DataHarvestRate     int           `name:"data_harvest_rate" default:"5" usage:"data earned per tick per program on a vault" json:"data_harvest_rate"`
	ComputeHarvestRate  int           `name:"compute_harvest_rate" default:"3" usage:"compute earned per tick per program on a relay" json:"compute_harvest_rate"`
	ProgramSpawnCost    int           `name:"program_spawn_cost" default:"12" usage:"data cost to spawn a program" json:"program_spawn_cost"`
	VirusDeployCost     int           `name:"virus_deploy_cost" default:"15" usage:"compute cost to deploy a virus" json:"virus_deploy_cost"`
	ProgramUpkeep       int           `name:"program_upkeep" default:"1" usage:"data cost per program per tick" json:"program_upkeep"`
	CoreHoldCost        int           `name:"core_hold_cost" default:"2" usage:"compute cost per program on core per tick" json:"core_hold_cost"`
	SurviveMin          int           `name:"survive_min" default:"1" usage:"min neighbor support for program survival" json:"survive_min"`
	SurviveMax          int           `name:"survive_max" default:"10" usage:"max neighbor support before overcrowding" json:"survive_max"`
	SpreadExact         int           `name:"spread_exact" default:"3" usage:"min neighbor programs for auto-spread" json:"spread_exact"`
	InitialData         int           `name:"initial_data" default:"150" usage:"starting data resource" json:"initial_data"`
	InitialCompute      int           `name:"initial_compute" default:"60" usage:"starting compute resource" json:"initial_compute"`
	ICESpawnTick        int           `name:"ice_spawn_tick" default:"25" usage:"tick when first new ICE spawns" json:"ice_spawn_tick"`
	ICESpawnMinInterval int           `name:"ice_spawn_min_interval" default:"8" usage:"fastest ICE spawn interval (tick floor)" json:"ice_spawn_min_interval"`
	ICEEscalationTick   int           `name:"ice_escalation_tick" default:"80" usage:"tick when ICE escalation starts" json:"ice_escalation_tick"`
	ICEEscalationRate   int           `name:"ice_escalation_rate" default:"50" usage:"ticks between ICE escalation bursts" json:"ice_escalation_rate"`
	EventLogSize        int           `name:"event_log_size" default:"20" usage:"events shown in snapshot" json:"event_log_size"`
	EventLogFile        string        `name:"event_log_file" default:"./cyberspace.log" usage:"file path to write event log" json:"event_log_file"`
	SaveDir             string        `name:"save_dir" default:"" usage:"directory for save files (default fallback path ~/.cyberspace/saves)" json:"save_dir"`
}
