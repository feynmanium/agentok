package edqlite

import "container/heap"

// priorityQueue is a heap-based queue that orders events by Priority (descending)
// then by CreatedAt (ascending, FIFO within same priority).
type priorityQueue struct {
	items eventHeap
}

func newPriorityQueue() *priorityQueue {
	pq := &priorityQueue{}
	heap.Init(&pq.items)
	return pq
}

// Push adds an event to the priority queue.
func (pq *priorityQueue) Push(evt Event) {
	heap.Push(&pq.items, evt)
}

// Pop removes and returns the highest-priority event.
// Returns false if the queue is empty.
func (pq *priorityQueue) Pop() (Event, bool) {
	if pq.items.Len() == 0 {
		return Event{}, false
	}
	evt := heap.Pop(&pq.items).(Event)
	return evt, true
}

// Len returns the number of events in the queue.
func (pq *priorityQueue) Len() int {
	return pq.items.Len()
}

// eventHeap implements heap.Interface for Event.
type eventHeap []Event

func (h eventHeap) Len() int { return len(h) }

func (h eventHeap) Less(i, j int) bool {
	// Higher priority first
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
	// Earlier timestamp first (FIFO within same priority)
	return h[i].CreatedAt.Before(h[j].CreatedAt)
}

func (h eventHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *eventHeap) Push(x any) {
	*h = append(*h, x.(Event))
}

func (h *eventHeap) Pop() any {
	old := *h
	n := len(old)
	evt := old[n-1]
	*h = old[:n-1]
	return evt
}
