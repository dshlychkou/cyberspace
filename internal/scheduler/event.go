package scheduler

type EventType int

const (
	EventICESpawn EventType = iota
	EventICEEscalation
)

type Event struct {
	Tick     int       `json:"tick"`
	Priority int       `json:"priority"`
	Type     EventType `json:"type"`
	Data     any       `json:"data,omitempty"`
}
