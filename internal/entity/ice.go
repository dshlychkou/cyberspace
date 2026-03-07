package entity

type ICEState int

const (
	ICEPatrolling ICEState = iota
	ICEPursuing
	ICESuppressing
)

type ICE struct {
	Entity
	State ICEState
}

func NewICE(id int, nodeID uint64) *ICE {
	return &ICE{
		Entity: Entity{
			ID:     id,
			Kind:   KindICE,
			NodeID: nodeID,
			Alive:  true,
		},
		State: ICEPatrolling,
	}
}
