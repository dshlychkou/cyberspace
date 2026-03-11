package network

import (
	"github.com/barnowlsnest/go-datalib/v5/pkg/dag"
)

const GroupName = "cyberspace"

type Network struct {
	Graph *dag.Graph
	Nodes map[uint64]*Node
}

func New() *Network {
	g := dag.New()
	_ = g.AddGroup(GroupName)
	return &Network{
		Graph: g,
		Nodes: make(map[uint64]*Node),
	}
}

func (n *Network) AddNode(node *Node) {
	gn := dag.GroupNode{ID: node.ID, Group: GroupName}
	_ = n.Graph.AddNode(gn)
	n.Nodes[node.ID] = node
}

func (n *Network) Connect(fromID, toID uint64) {
	from := dag.GroupNode{ID: fromID, Group: GroupName}
	to := dag.GroupNode{ID: toID, Group: GroupName}
	_ = n.Graph.AddEdge(from, to)
	_ = n.Graph.AddEdge(to, from) // undirected
}

func (n *Network) GetNode(id uint64) *Node {
	return n.Nodes[id]
}

func (n *Network) Neighbors(nodeID uint64) []uint64 {
	var neighbors []uint64
	gn := dag.GroupNode{ID: nodeID, Group: GroupName}
	_ = n.Graph.ForEachNeighbour(gn, func(edge dag.AdjacencyEdge, err error) {
		if err == nil {
			neighbors = append(neighbors, edge.To)
		}
	})
	return neighbors
}

func (n *Network) NodeIDs() []uint64 {
	ids := make([]uint64, 0, len(n.Nodes))
	for id := range n.Nodes {
		ids = append(ids, id)
	}
	return ids
}

func (n *Network) NodesByType(t NodeType) []*Node {
	var nodes []*Node
	for _, node := range n.Nodes {
		if node.Type == t {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
