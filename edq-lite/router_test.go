package edqlite

import (
	"context"
	"testing"
	"time"
)

func TestTopicRouterDirectRoute(t *testing.T) {
	b := NewBroker()
	r := NewTopicRouter(b)

	r.AddRoute("alice", "bob", "chat.1.agent.bob")

	evt := Event{Sender: "alice", Receiver: "bob"}
	topic := r.RouteEvent(evt)
	if topic != "chat.1.agent.bob" {
		t.Errorf("RouteEvent() = %q, want %q", topic, "chat.1.agent.bob")
	}
}

func TestTopicRouterBroadcastRoute(t *testing.T) {
	b := NewBroker()
	r := NewTopicRouter(b)

	r.AddBroadcastRoute("alice", "chat.1.broadcast")

	evt := Event{Sender: "alice", Receiver: ""}
	topic := r.RouteEvent(evt)
	if topic != "chat.1.broadcast" {
		t.Errorf("RouteEvent() = %q, want %q", topic, "chat.1.broadcast")
	}
}

func TestTopicRouterDefaultRoute(t *testing.T) {
	b := NewBroker()
	r := NewTopicRouter(b)

	// No routes configured
	evt := Event{Sender: "alice", Receiver: "bob"}
	topic := r.RouteEvent(evt)
	if topic != "chat.default.agent.bob" {
		t.Errorf("RouteEvent() = %q, want %q", topic, "chat.default.agent.bob")
	}

	// No receiver
	evt2 := Event{Sender: "alice"}
	topic2 := r.RouteEvent(evt2)
	if topic2 != "chat.default.broadcast" {
		t.Errorf("RouteEvent() = %q, want %q", topic2, "chat.default.broadcast")
	}
}

func TestTopicRouterLoadFromEdges(t *testing.T) {
	b := NewBroker()
	r := NewTopicRouter(b)

	edges := []Edge{
		{Source: "user_proxy", Target: "assistant"},
		{Source: "assistant", Target: "code_reviewer"},
	}

	if err := r.LoadFromEdges("42", edges); err != nil {
		t.Fatalf("LoadFromEdges() error = %v", err)
	}

	// Forward route
	evt := Event{Sender: "user_proxy", Receiver: "assistant"}
	if topic := r.RouteEvent(evt); topic != "chat.42.agent.assistant" {
		t.Errorf("forward route = %q, want %q", topic, "chat.42.agent.assistant")
	}

	// Reverse route (bidirectional)
	evt2 := Event{Sender: "assistant", Receiver: "user_proxy"}
	if topic := r.RouteEvent(evt2); topic != "chat.42.agent.user_proxy" {
		t.Errorf("reverse route = %q, want %q", topic, "chat.42.agent.user_proxy")
	}
}

func TestTopicRouterRouteAndPublish(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(16))
	ctx := context.Background()
	r := NewTopicRouter(b)

	r.AddRoute("alice", "bob", "chat.1.agent.bob")

	received := make(chan Event, 1)
	b.Subscribe("chat.1.agent.bob", func(evt Event) error {
		received <- evt
		return nil
	})

	b.Start(ctx)
	defer b.Shutdown(ctx)

	evt := NewEvent("", "user", "alice", "bob", "routed message")
	if err := r.RouteAndPublish(ctx, evt); err != nil {
		t.Fatalf("RouteAndPublish() error = %v", err)
	}

	select {
	case got := <-received:
		if got.Content != "routed message" {
			t.Errorf("content = %q, want %q", got.Content, "routed message")
		}
		if got.Topic != "chat.1.agent.bob" {
			t.Errorf("topic = %q, want %q", got.Topic, "chat.1.agent.bob")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestTopicRouterInvalidTopic(t *testing.T) {
	b := NewBroker()
	r := NewTopicRouter(b)

	if err := r.AddRoute("a", "b", ""); err == nil {
		t.Error("expected error for empty topic")
	}
	if err := r.AddBroadcastRoute("a", ""); err == nil {
		t.Error("expected error for empty broadcast topic")
	}
}
