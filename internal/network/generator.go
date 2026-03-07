package network

import (
	"math/rand/v2"
)

func Generate(rng *rand.Rand) *Network {
	net := New()

	// Create core (1 node)
	coreID := 1
	net.AddNode(NewNode(coreID, NodeCore))

	// Create firewalls around core (2-3 nodes)
	fwCount := 2 + rng.IntN(2)
	fwIDs := make([]int, fwCount)
	nextID := 2
	for i := range fwCount {
		fwIDs[i] = nextID
		net.AddNode(NewNode(nextID, NodeFirewall))
		net.Connect(coreID, nextID)
		nextID++
	}

	// Create servers (3-4 nodes)
	srvCount := 3 + rng.IntN(2)
	srvIDs := make([]int, srvCount)
	for i := range srvCount {
		srvIDs[i] = nextID
		net.AddNode(NewNode(nextID, NodeServer))
		// Connect to a random firewall
		fwIdx := rng.IntN(len(fwIDs))
		net.Connect(nextID, fwIDs[fwIdx])
		nextID++
	}

	// Create relays (2-3 nodes) connecting servers
	relCount := 2 + rng.IntN(2)
	relIDs := make([]int, relCount)
	for i := range relCount {
		relIDs[i] = nextID
		net.AddNode(NewNode(nextID, NodeRelay))
		// Connect to two random servers
		s1 := rng.IntN(len(srvIDs))
		s2 := (s1 + 1 + rng.IntN(max(1, len(srvIDs)-1))) % len(srvIDs)
		net.Connect(nextID, srvIDs[s1])
		net.Connect(nextID, srvIDs[s2])
		nextID++
	}

	// Create vaults (1-2 data nodes)
	vaultCount := 1 + rng.IntN(2)
	for range vaultCount {
		net.AddNode(NewNode(nextID, NodeVault))
		// Connect to a random server and relay
		net.Connect(nextID, srvIDs[rng.IntN(len(srvIDs))])
		if len(relIDs) > 0 {
			net.Connect(nextID, relIDs[rng.IntN(len(relIDs))])
		}
		nextID++
	}

	// Add a few extra cross-connections for interesting topology
	allIDs := net.NodeIDs()
	extraEdges := 2 + rng.IntN(3)
	for range extraEdges {
		a := allIDs[rng.IntN(len(allIDs))]
		b := allIDs[rng.IntN(len(allIDs))]
		if a != b {
			net.Connect(int(a), int(b))
		}
	}

	return net
}
