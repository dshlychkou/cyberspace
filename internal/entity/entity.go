package entity

type Entity struct {
	ID     int    `json:"id"`
	Kind   Kind   `json:"kind"`
	NodeID uint64 `json:"node_id"`
	Age    int    `json:"age"`
	Alive  bool   `json:"alive"`
	MaxAge int    `json:"max_age"`
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
