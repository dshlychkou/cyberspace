package scheduler

type EventType int

const (
	EventICESpawn EventType = iota
	EventICEEscalation
)

type Event struct {
	Tick     int
	Priority int // lower = higher priority
	Type     EventType
	Data     any
}
