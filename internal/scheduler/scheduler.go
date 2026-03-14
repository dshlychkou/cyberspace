package scheduler

import (
	"github.com/barnowlsnest/go-datalib/v5/pkg/tree"
)

type Scheduler struct {
	heap *tree.Heap[Event]
}

func New() *Scheduler {
	return &Scheduler{
		heap: tree.NewHeap(func(a, b Event) bool {
			if a.Tick != b.Tick {
				return a.Tick < b.Tick
			}
			return a.Priority < b.Priority
		}),
	}
}

func (s *Scheduler) Schedule(e Event) {
	s.heap.Push(e)
}

func (s *Scheduler) DueEvents(currentTick int) []Event {
	var events []Event
	for {
		e, ok := s.heap.Peek()
		if !ok || e.Tick > currentTick {
			break
		}
		e, _ = s.heap.Pop()
		events = append(events, e)
	}
	return events
}

func (s *Scheduler) Size() int {
	return s.heap.Size()
}

func (s *Scheduler) PendingEvents() []Event {
	return s.heap.ToSlice()
}

func Restore(events []Event) *Scheduler {
	return &Scheduler{
		heap: tree.HeapFromSlice(events, func(a, b Event) bool {
			if a.Tick != b.Tick {
				return a.Tick < b.Tick
			}
			return a.Priority < b.Priority
		}),
	}
}
