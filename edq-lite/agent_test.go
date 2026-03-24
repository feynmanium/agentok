package edqlite

import (
	"context"
	"testing"
	"time"
)

func TestAgentMailboxSendReceive(t *testing.T) {
	b := NewBroker(WithWorkers(2), WithQueueDepth(64))
	ctx := context.Background()

	alice, err := NewAgentMailbox(b, "alice", AgentTypeUserProxy, WithAgentChatID("1"))
	if err != nil {
		t.Fatalf("NewAgentMailbox(alice) error = %v", err)
	}
	defer alice.Close()

	bob, err := NewAgentMailbox(b, "bob", AgentTypeAssistant, WithAgentChatID("1"))
	if err != nil {
		t.Fatalf("NewAgentMailbox(bob) error = %v", err)
	}
	defer bob.Close()

	if err := b.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer b.Shutdown(ctx)

	// Alice sends to Bob
	if err := alice.Send(ctx, "bob", "hello bob", WithSendType("user")); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	select {
	case evt := <-bob.Receive():
		if evt.Content != "hello bob" {
			t.Errorf("content = %q, want %q", evt.Content, "hello bob")
		}
		if evt.Sender != "alice" {
			t.Errorf("sender = %q, want %q", evt.Sender, "alice")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestAgentMailboxBroadcast(t *testing.T) {
	b := NewBroker(WithWorkers(2), WithQueueDepth(64))
	ctx := context.Background()

	alice, _ := NewAgentMailbox(b, "alice", AgentTypeUserProxy, WithAgentChatID("1"))
	defer alice.Close()
	bob, _ := NewAgentMailbox(b, "bob", AgentTypeAssistant, WithAgentChatID("1"))
	defer bob.Close()
	charlie, _ := NewAgentMailbox(b, "charlie", AgentTypeAssistant, WithAgentChatID("1"))
	defer charlie.Close()

	b.Start(ctx)
	defer b.Shutdown(ctx)

	// Alice broadcasts
	alice.Send(ctx, "", "hello everyone", WithSendType("user"))

	// Both bob and charlie should receive (alice also subscribed to broadcast)
	received := 0
	timeout := time.After(2 * time.Second)
	for received < 2 {
		select {
		case <-bob.Receive():
			received++
		case <-charlie.Receive():
			received++
		case <-timeout:
			t.Fatalf("timeout: received only %d of 2 messages", received)
		}
	}
}

func TestAgentMailboxName(t *testing.T) {
	b := NewBroker()
	mb, _ := NewAgentMailbox(b, "test-agent", AgentTypeConversable, WithAgentChatID("1"))
	defer mb.Close()

	if mb.Name() != "test-agent" {
		t.Errorf("Name() = %q, want %q", mb.Name(), "test-agent")
	}
	if mb.Type() != AgentTypeConversable {
		t.Errorf("Type() = %q, want %q", mb.Type(), AgentTypeConversable)
	}
}

func TestAgentMailboxClose(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(16))
	ctx := context.Background()

	mb, _ := NewAgentMailbox(b, "closeme", AgentTypeAssistant, WithAgentChatID("1"))

	b.Start(ctx)
	defer b.Shutdown(ctx)

	// Close should be idempotent
	mb.Close()
	mb.Close()

	// Channel should be closed
	_, ok := <-mb.Receive()
	if ok {
		t.Error("Receive() should be closed after Close()")
	}
}

func TestAgentMailboxWithSendOptions(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(64))
	ctx := context.Background()

	received := make(chan Event, 1)
	b.Subscribe("chat.1.agent.bob", func(evt Event) error {
		received <- evt
		return nil
	})

	alice, _ := NewAgentMailbox(b, "alice", AgentTypeUserProxy, WithAgentChatID("1"))
	defer alice.Close()

	b.Start(ctx)
	defer b.Shutdown(ctx)

	meta := map[string]any{"tool": "search"}
	alice.Send(ctx, "bob", "search for me",
		WithSendType("tool_call"),
		WithSendPriority(PriorityHigh),
		WithSendMetadata(meta),
	)

	select {
	case evt := <-received:
		if evt.Type != "tool_call" {
			t.Errorf("type = %q, want %q", evt.Type, "tool_call")
		}
		if evt.Priority != PriorityHigh {
			t.Errorf("priority = %d, want %d", evt.Priority, PriorityHigh)
		}
		if evt.Metadata["tool"] != "search" {
			t.Errorf("metadata[tool] = %v, want %q", evt.Metadata["tool"], "search")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
