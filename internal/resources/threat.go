package resources

import (
	"github.com/barnowlsnest/go-datalib/pkg/tree"
)

type ThreatTracker struct {
	fenwick *tree.Fenwick[int]
	size    int
}

func NewThreatTracker(regionCount int) *ThreatTracker {
	return &ThreatTracker{
		fenwick: tree.NewFenwick[int](regionCount),
		size:    regionCount,
	}
}

func (t *ThreatTracker) UpdateThreat(region int, delta int) {
	if region >= 1 && region <= t.size {
		t.fenwick.Update(region, delta)
	}
}

func (t *ThreatTracker) GetThreat(region int) int {
	if region >= 1 && region <= t.size {
		return t.fenwick.Get(region)
	}
	return 0
}

func (t *ThreatTracker) TotalThreat() int {
	return t.fenwick.Query(t.size)
}

func (t *ThreatTracker) RegionThreat(from, to int) int {
	return t.fenwick.RangeQuery(from, to)
}
