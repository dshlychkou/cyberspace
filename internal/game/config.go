package game

import "time"

type Config struct {
	TickRate          time.Duration
	InitialPrograms   int
	InitialICE        int
	ICEEscalationRate float64 // ICE spawn rate multiplier per 100 ticks
	VirusLifespan     int     // ticks before virus decays
	FirewallDelay     int     // ticks to pass through firewall
	FirewallOverride  int     // adjacent programs needed to deactivate firewall
	CoreWinThreshold  int     // programs needed on core
	CoreWinDuration   int     // consecutive ticks to win
	DataHarvestRate   int     // resources per tick per program on data node
	ProgramSpawnCost  int     // resource cost to spawn a program
	VirusDeployCost   int     // resource cost to deploy a virus
	SurviveMin        int     // min neighbors to survive
	SurviveMax        int     // max neighbors to survive
	SpreadExact       int     // min neighbor count to auto-spread
}

func DefaultConfig() Config {
	return Config{
		TickRate:          1000 * time.Millisecond,
		InitialPrograms:   3,
		InitialICE:        2,
		ICEEscalationRate: 1.2,
		VirusLifespan:     8,
		FirewallDelay:     2,
		FirewallOverride:  4,
		CoreWinThreshold:  2,
		CoreWinDuration:   10,
		DataHarvestRate:   5,
		ProgramSpawnCost:  15,
		VirusDeployCost:   25,
		SurviveMin:        1,
		SurviveMax:        6,
		SpreadExact:       2,
	}
}
