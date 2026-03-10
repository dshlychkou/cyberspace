package network

import "fmt"

const (
	symbolFilledDiamond = "\u25c6" // ◆
	symbolHollowDiamond = "\u25c7" // ◇
	symbolStar          = "\u2605" // ★
)

type NodeType int

const (
	NodeServer NodeType = iota
	NodeVault
	NodeRelay
	NodeFirewall
	NodeCore
)

func (t NodeType) String() string {
	switch t {
	case NodeServer:
		return "SRV"
	case NodeVault:
		return "VLT"
	case NodeRelay:
		return "REL"
	case NodeFirewall:
		return "FWL"
	case NodeCore:
		return "CORE"
	default:
		return "???"
	}
}

func (t NodeType) Symbol() string {
	switch t {
	case NodeServer:
		return symbolFilledDiamond
	case NodeVault:
		return symbolFilledDiamond
	case NodeRelay:
		return symbolHollowDiamond
	case NodeFirewall:
		return symbolFilledDiamond
	case NodeCore:
		return symbolStar
	default:
		return "?"
	}
}

type Node struct {
	ID       uint64
	Type     NodeType
	Label    string
	Entities []int // entity IDs present on this node
}

func NewNode(id uint64, nodeType NodeType) *Node {
	return &Node{
		ID:       id,
		Type:     nodeType,
		Label:    fmt.Sprintf("%s-%d", nodeType, id),
		Entities: make([]int, 0),
	}
}

func (n *Node) HasEntity(entityID int) bool {
	for _, eid := range n.Entities {
		if eid == entityID {
			return true
		}
	}
	return false
}

func (n *Node) AddEntity(entityID int) {
	if !n.HasEntity(entityID) {
		n.Entities = append(n.Entities, entityID)
	}
}

func (n *Node) RemoveEntity(entityID int) {
	for i, eid := range n.Entities {
		if eid == entityID {
			n.Entities = append(n.Entities[:i], n.Entities[i+1:]...)
			return
		}
	}
}
