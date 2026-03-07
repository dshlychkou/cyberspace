package resources

import (
	"github.com/barnowlsnest/go-datalib/pkg/tree"
)

type ResourceType string

const (
	ResData    ResourceType = "Data"
	ResCompute ResourceType = "Compute"
	ResCycles  ResourceType = "Cycles"
)

type Tracker struct {
	btree *tree.BTree[string, int]
}

func NewTracker() *Tracker {
	t := &Tracker{
		btree: tree.NewBTree[string, int](3),
	}
	t.btree.Insert(string(ResData), 0)
	t.btree.Insert(string(ResCompute), 0)
	t.btree.Insert(string(ResCycles), 0)
	return t
}

func (t *Tracker) Add(res ResourceType, amount int) {
	current, found := t.btree.Search(string(res))
	if !found {
		current = 0
	}
	t.btree.Insert(string(res), current+amount)
}

func (t *Tracker) Get(res ResourceType) int {
	val, found := t.btree.Search(string(res))
	if !found {
		return 0
	}
	return val
}

func (t *Tracker) Spend(res ResourceType, amount int) bool {
	current := t.Get(res)
	if current < amount {
		return false
	}
	t.btree.Insert(string(res), current-amount)
	return true
}
