package entity

type Virus struct {
	Entity
}

func NewVirus(id int, nodeID uint64, lifespan int) *Virus {
	return &Virus{
		Entity: Entity{
			ID:     id,
			Kind:   KindVirus,
			NodeID: nodeID,
			Alive:  true,
			MaxAge: lifespan,
		},
	}
}
