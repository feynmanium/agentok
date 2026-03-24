package edqlite

import (
	"testing"
	"time"
)

func TestPriorityQueueOrdering(t *testing.T) {
	pq := newPriorityQueue()
	now := time.Now()

	// Add events with different priorities
	events := []Event{
		{ID: "low", Priority: PriorityLow, CreatedAt: now},
		{ID: "urgent", Priority: PriorityUrgent, CreatedAt: now},
		{ID: "normal", Priority: PriorityNormal, CreatedAt: now},
		{ID: "high", Priority: PriorityHigh, CreatedAt: now},
	}

	for _, e := range events {
		pq.Push(e)
	}

	// Should come out in priority order: urgent, high, normal, low
	expected := []string{"urgent", "high", "normal", "low"}
	for _, wantID := range expected {
		evt, ok := pq.Pop()
		if !ok {
			t.Fatal("unexpected empty queue")
		}
		if evt.ID != wantID {
			t.Errorf("got ID %q, want %q", evt.ID, wantID)
		}
	}

	if pq.Len() != 0 {
		t.Errorf("queue should be empty, has %d items", pq.Len())
	}
}

func TestPriorityQueueFIFOWithinSamePriority(t *testing.T) {
	pq := newPriorityQueue()
	base := time.Now()

	// Same priority, different timestamps
	pq.Push(Event{ID: "third", Priority: PriorityNormal, CreatedAt: base.Add(2 * time.Second)})
	pq.Push(Event{ID: "first", Priority: PriorityNormal, CreatedAt: base})
	pq.Push(Event{ID: "second", Priority: PriorityNormal, CreatedAt: base.Add(1 * time.Second)})

	expected := []string{"first", "second", "third"}
	for _, wantID := range expected {
		evt, ok := pq.Pop()
		if !ok {
			t.Fatal("unexpected empty queue")
		}
		if evt.ID != wantID {
			t.Errorf("got ID %q, want %q", evt.ID, wantID)
		}
	}
}

func TestPriorityQueueEmpty(t *testing.T) {
	pq := newPriorityQueue()
	_, ok := pq.Pop()
	if ok {
		t.Error("Pop on empty queue should return false")
	}
	if pq.Len() != 0 {
		t.Errorf("Len() = %d, want 0", pq.Len())
	}
}

func TestPriorityQueueLen(t *testing.T) {
	pq := newPriorityQueue()
	for i := 0; i < 5; i++ {
		pq.Push(Event{ID: string(rune('a' + i)), Priority: PriorityNormal, CreatedAt: time.Now()})
	}
	if pq.Len() != 5 {
		t.Errorf("Len() = %d, want 5", pq.Len())
	}
	pq.Pop()
	if pq.Len() != 4 {
		t.Errorf("Len() = %d after pop, want 4", pq.Len())
	}
}
