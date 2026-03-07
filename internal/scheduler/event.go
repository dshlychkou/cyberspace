package scheduler

type EventType int

const (
	EventICESpawn EventType = iota
	EventICEEscalation
	EventVirusDecay
)

type Event struct {
	Tick     int
	Priority int // lower = higher priority
	Type     EventType
	Data     any
}
