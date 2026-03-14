package entity

type ProgramState int

const (
	ProgramSpreading ProgramState = iota
)

type Program struct {
	Entity
	State ProgramState `json:"state"`
}

func NewProgram(id int, nodeID uint64) *Program {
	return &Program{
		Entity: Entity{
			ID:     id,
			Kind:   KindProgram,
			NodeID: nodeID,
			Alive:  true,
		},
		State: ProgramSpreading,
	}
}
