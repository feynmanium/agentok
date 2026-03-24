package edqlite

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBrokerStartStop(t *testing.T) {
	b := NewBroker(WithWorkers(2), WithQueueDepth(16))
	ctx := context.Background()

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Double start should fail
	if err := b.Start(ctx); err == nil {
		t.Error("expected error on double Start()")
	}

	if err := b.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestBrokerPublishSubscribe(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(16))
	ctx := context.Background()

	received := make(chan Event, 1)
	_, err := b.Subscribe("chat.1.agent.bob", func(evt Event) error {
		received <- evt
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer b.Shutdown(ctx)

	evt := NewEvent("chat.1.agent.bob", "user", "alice", "bob", "hello")
	if err := b.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	select {
	case got := <-received:
		if got.Content != "hello" {
			t.Errorf("content = %q, want %q", got.Content, "hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestBrokerWildcardSubscription(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(16))
	ctx := context.Background()

	var count atomic.Int32
	_, err := b.Subscribe("chat.1.agent.*", func(evt Event) error {
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer b.Shutdown(ctx)

	// Publish to different agents in the same chat
	for _, name := range []string{"bob", "alice", "charlie"} {
		evt := NewEvent("chat.1.agent."+name, "user", "sys", name, "hi")
		if err := b.Publish(ctx, evt); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	// Wait for delivery
	time.Sleep(200 * time.Millisecond)

	if got := count.Load(); got != 3 {
		t.Errorf("received %d events, want 3", got)
	}
}

func TestBrokerMultipleSubscribers(t *testing.T) {
	b := NewBroker(WithWorkers(2), WithQueueDepth(16))
	ctx := context.Background()

	var count1, count2 atomic.Int32

	_, _ = b.Subscribe("chat.1.broadcast", func(evt Event) error {
		count1.Add(1)
		return nil
	})
	_, _ = b.Subscribe("chat.1.broadcast", func(evt Event) error {
		count2.Add(1)
		return nil
	})

	b.Start(ctx)
	defer b.Shutdown(ctx)

	evt := NewEvent("chat.1.broadcast", "system", "sys", "", "notify")
	b.Publish(ctx, evt)

	time.Sleep(200 * time.Millisecond)

	if count1.Load() != 1 || count2.Load() != 1 {
		t.Errorf("both subscribers should receive the event: sub1=%d, sub2=%d", count1.Load(), count2.Load())
	}
}

func TestBrokerUnsubscribe(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(16))
	ctx := context.Background()

	var count atomic.Int32
	subID, _ := b.Subscribe("chat.1.agent.bob", func(evt Event) error {
		count.Add(1)
		return nil
	})

	b.Start(ctx)
	defer b.Shutdown(ctx)

	// Publish once
	b.Publish(ctx, NewEvent("chat.1.agent.bob", "user", "alice", "bob", "msg1"))
	time.Sleep(100 * time.Millisecond)

	// Unsubscribe
	if err := b.Unsubscribe(subID); err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}

	// Publish again - should not be received
	b.Publish(ctx, NewEvent("chat.1.agent.bob", "user", "alice", "bob", "msg2"))
	time.Sleep(100 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("received %d events after unsubscribe, want 1", got)
	}
}

func TestBrokerConcurrentPublish(t *testing.T) {
	b := NewBroker(WithWorkers(4), WithQueueDepth(256))
	ctx := context.Background()

	var total atomic.Int64
	_, _ = b.Subscribe(">", func(evt Event) error {
		total.Add(1)
		return nil
	})

	b.Start(ctx)
	defer b.Shutdown(ctx)

	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			evt := NewEvent("chat.1.agent.bob", "user", "alice", "bob", "msg")
			b.Publish(ctx, evt)
		}(i)
	}
	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	if got := total.Load(); got != int64(n) {
		t.Errorf("received %d events, want %d", got, n)
	}
}

func TestBrokerDeadLetter(t *testing.T) {
	var deadLettered atomic.Int32
	b := NewBroker(
		WithWorkers(1),
		WithQueueDepth(16),
		WithDeadLetterHandler(func(evt Event) error {
			deadLettered.Add(1)
			return nil
		}),
	)
	ctx := context.Background()
	b.Start(ctx)
	defer b.Shutdown(ctx)

	// Publish with no subscribers
	b.Publish(ctx, NewEvent("chat.1.agent.nobody", "user", "alice", "nobody", "hello"))
	time.Sleep(200 * time.Millisecond)

	if got := deadLettered.Load(); got != 1 {
		t.Errorf("dead lettered %d, want 1", got)
	}
}

func TestBrokerPublishBeforeStart(t *testing.T) {
	b := NewBroker()
	evt := NewEvent("chat.1.agent.bob", "user", "alice", "bob", "hello")
	err := b.Publish(context.Background(), evt)
	if err == nil {
		t.Error("expected error publishing before Start()")
	}
}

func TestBrokerPublishInvalidEvent(t *testing.T) {
	b := NewBroker(WithWorkers(1))
	ctx := context.Background()
	b.Start(ctx)
	defer b.Shutdown(ctx)

	err := b.Publish(ctx, Event{}) // Invalid: missing required fields
	if err == nil {
		t.Error("expected error for invalid event")
	}
}

func TestBrokerTopics(t *testing.T) {
	b := NewBroker()
	_, _ = b.Subscribe("chat.1.agent.*", func(Event) error { return nil })
	_, _ = b.Subscribe("chat.1.agent.*", func(Event) error { return nil })
	_, _ = b.Subscribe("chat.1.broadcast", func(Event) error { return nil })

	topics := b.Topics()
	if len(topics) != 2 {
		t.Errorf("got %d topic groups, want 2", len(topics))
	}
}

func TestBrokerSubscribeNilHandler(t *testing.T) {
	b := NewBroker()
	_, err := b.Subscribe("chat.1.>", nil)
	if err == nil {
		t.Error("expected error for nil handler")
	}
}

func TestBrokerSubscribeInvalidPattern(t *testing.T) {
	b := NewBroker()
	_, err := b.Subscribe("", func(Event) error { return nil })
	if err == nil {
		t.Error("expected error for empty pattern")
	}
}

func TestBrokerSubscribeDuplicateID(t *testing.T) {
	b := NewBroker()
	_, err := b.Subscribe("a.b", func(Event) error { return nil }, WithSubscriptionID("dup"))
	if err != nil {
		t.Fatalf("first subscribe error = %v", err)
	}
	_, err = b.Subscribe("a.b", func(Event) error { return nil }, WithSubscriptionID("dup"))
	if err == nil {
		t.Error("expected error for duplicate subscription ID")
	}
}

func TestBrokerStats(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(16))
	ctx := context.Background()
	_, _ = b.Subscribe("chat.1.>", func(Event) error { return nil })
	b.Start(ctx)
	defer b.Shutdown(ctx)

	b.Publish(ctx, NewEvent("chat.1.agent.bob", "user", "alice", "bob", "hi"))
	time.Sleep(200 * time.Millisecond)

	stats := b.Stats()
	if stats.Published != 1 {
		t.Errorf("published = %d, want 1", stats.Published)
	}
	if stats.Delivered != 1 {
		t.Errorf("delivered = %d, want 1", stats.Delivered)
	}
}
