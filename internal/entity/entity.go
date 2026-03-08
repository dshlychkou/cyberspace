package entity

type Entity struct {
	ID     int
	Kind   Kind
	NodeID uint64
	Age    int // ticks alive
	Alive  bool
	MaxAge int // 0 = infinite
}

func NewEntity(id int, kind Kind, nodeID uint64) *Entity {
	return &Entity{
		ID:     id,
		Kind:   kind,
		NodeID: nodeID,
		Alive:  true,
	}
}

func (e *Entity) Tick() {
	e.Age++
	if e.MaxAge > 0 && e.Age >= e.MaxAge {
		e.Alive = false
	}
}
