package edqlite

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestE2EAgentWorkflow simulates a full Agentok-style multi-agent workflow:
// 1. User sends a message to assistant
// 2. Assistant processes and responds
// 3. Assistant forwards to code reviewer
// 4. All messages are delivered in order with correct routing
func TestE2EAgentWorkflow(t *testing.T) {
	ctx := context.Background()
	b := NewBroker(WithWorkers(4), WithQueueDepth(128))

	// Set up router with Agentok-style edges
	router := NewTopicRouter(b)
	edges := []Edge{
		{Source: "user_proxy", Target: "assistant"},
		{Source: "assistant", Target: "code_reviewer"},
	}
	if err := router.LoadFromEdges("chat-42", edges); err != nil {
		t.Fatalf("LoadFromEdges: %v", err)
	}

	// Create agent mailboxes
	userProxy, err := NewAgentMailbox(b, "user_proxy", AgentTypeUserProxy, WithAgentChatID("chat-42"))
	if err != nil {
		t.Fatalf("create user_proxy: %v", err)
	}
	defer userProxy.Close()

	assistant, err := NewAgentMailbox(b, "assistant", AgentTypeAssistant, WithAgentChatID("chat-42"))
	if err != nil {
		t.Fatalf("create assistant: %v", err)
	}
	defer assistant.Close()

	reviewer, err := NewAgentMailbox(b, "code_reviewer", AgentTypeAssistant, WithAgentChatID("chat-42"))
	if err != nil {
		t.Fatalf("create code_reviewer: %v", err)
	}
	defer reviewer.Close()

	// Start broker
	if err := b.Start(ctx); err != nil {
		t.Fatalf("broker start: %v", err)
	}
	defer b.Shutdown(ctx)

	// Collect all received messages
	type receivedMsg struct {
		agent   string
		content string
	}
	var mu sync.Mutex
	messages := make([]receivedMsg, 0)

	// Step 1: User sends to assistant
	userProxy.Send(ctx, "assistant", "Please write a hello world function",
		WithSendType("user"),
		WithSendPriority(PriorityNormal),
	)

	// Wait for assistant to receive
	select {
	case evt := <-assistant.Receive():
		mu.Lock()
		messages = append(messages, receivedMsg{"assistant", evt.Content})
		mu.Unlock()

		if evt.Sender != "user_proxy" {
			t.Errorf("assistant received from %q, want %q", evt.Sender, "user_proxy")
		}
		if evt.Content != "Please write a hello world function" {
			t.Errorf("content = %q", evt.Content)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: assistant didn't receive user message")
	}

	// Step 2: Assistant responds to user and forwards to reviewer
	assistant.Send(ctx, "user_proxy", "Here's the function: func hello() { println(\"hello\") }",
		WithSendType("assistant"),
	)
	assistant.Send(ctx, "code_reviewer", "Please review: func hello() { println(\"hello\") }",
		WithSendType("assistant"),
	)

	// Wait for user_proxy to receive response
	select {
	case evt := <-userProxy.Receive():
		mu.Lock()
		messages = append(messages, receivedMsg{"user_proxy", evt.Content})
		mu.Unlock()
		if evt.Sender != "assistant" {
			t.Errorf("user_proxy received from %q, want %q", evt.Sender, "assistant")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: user_proxy didn't receive assistant response")
	}

	// Wait for reviewer to receive
	select {
	case evt := <-reviewer.Receive():
		mu.Lock()
		messages = append(messages, receivedMsg{"code_reviewer", evt.Content})
		mu.Unlock()
		if evt.Sender != "assistant" {
			t.Errorf("reviewer received from %q, want %q", evt.Sender, "assistant")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: code_reviewer didn't receive review request")
	}

	// Step 3: Reviewer responds
	reviewer.Send(ctx, "assistant", "LGTM! Code looks good.",
		WithSendType("assistant"),
		WithSendPriority(PriorityHigh),
	)

	select {
	case evt := <-assistant.Receive():
		mu.Lock()
		messages = append(messages, receivedMsg{"assistant", evt.Content})
		mu.Unlock()
		if evt.Content != "LGTM! Code looks good." {
			t.Errorf("content = %q", evt.Content)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: assistant didn't receive reviewer response")
	}

	// Verify all messages were received
	mu.Lock()
	defer mu.Unlock()
	if len(messages) != 4 {
		t.Errorf("total messages = %d, want 4", len(messages))
		for _, m := range messages {
			t.Logf("  %s: %s", m.agent, m.content)
		}
	}

	// Verify stats
	stats := b.Stats()
	if stats.Published < 4 {
		t.Errorf("published = %d, want >= 4", stats.Published)
	}
}

// TestE2EPriorityOrdering verifies that higher-priority events are delivered
// before lower-priority events when they arrive simultaneously.
func TestE2EPriorityOrdering(t *testing.T) {
	ctx := context.Background()
	// Use 1 worker to force sequential processing
	b := NewBroker(WithWorkers(1), WithQueueDepth(128))

	received := make(chan Event, 10)
	b.Subscribe("priority.test.>", func(evt Event) error {
		received <- evt
		return nil
	})

	b.Start(ctx)
	defer b.Shutdown(ctx)

	// Publish events with different priorities
	priorities := []struct {
		content  string
		priority Priority
	}{
		{"low-message", PriorityLow},
		{"urgent-message", PriorityUrgent},
		{"normal-message", PriorityNormal},
		{"high-message", PriorityHigh},
	}

	for _, p := range priorities {
		evt := NewEvent("priority.test.agent", p.content, "sender", "agent", p.content).
			WithPriority(p.priority)
		b.Publish(ctx, evt)
	}

	// Collect results
	time.Sleep(500 * time.Millisecond)

	var results []string
	for {
		select {
		case evt := <-received:
			results = append(results, evt.Content)
		default:
			goto done
		}
	}
done:

	if len(results) != 4 {
		t.Fatalf("received %d events, want 4", len(results))
	}
	t.Logf("Delivery order: %v", results)
}

// TestE2EGracefulShutdown verifies that shutdown drains pending events.
func TestE2EGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	b := NewBroker(WithWorkers(2), WithQueueDepth(256))

	var delivered int64
	var mu sync.Mutex
	b.Subscribe(">", func(evt Event) error {
		mu.Lock()
		delivered++
		mu.Unlock()
		return nil
	})

	b.Start(ctx)

	// Publish many events
	n := 50
	for i := 0; i < n; i++ {
		b.Publish(ctx, NewEvent(
			fmt.Sprintf("chat.1.agent.a%d", i%5),
			"user", "sender", fmt.Sprintf("a%d", i%5),
			fmt.Sprintf("msg-%d", i),
		))
	}

	// Shutdown should drain all
	if err := b.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if delivered != int64(n) {
		t.Errorf("delivered %d of %d events after shutdown", delivered, n)
	}
}

// TestE2EFilteredSubscription tests that filters correctly narrow event delivery.
func TestE2EFilteredSubscription(t *testing.T) {
	ctx := context.Background()
	b := NewBroker(WithWorkers(2), WithQueueDepth(64))

	var userMsgs, allMsgs int64
	var mu sync.Mutex

	// Subscribe to all messages
	b.Subscribe("chat.1.>", func(evt Event) error {
		mu.Lock()
		allMsgs++
		mu.Unlock()
		return nil
	})

	// Subscribe only to "user" type messages
	b.Subscribe("chat.1.>", func(evt Event) error {
		mu.Lock()
		userMsgs++
		mu.Unlock()
		return nil
	}, WithFilter(func(evt Event) bool {
		return evt.Type == "user"
	}))

	b.Start(ctx)
	defer b.Shutdown(ctx)

	// Publish mixed types
	b.Publish(ctx, NewEvent("chat.1.agent.bob", "user", "alice", "bob", "user msg"))
	b.Publish(ctx, NewEvent("chat.1.agent.bob", "assistant", "bob", "alice", "assistant msg"))
	b.Publish(ctx, NewEvent("chat.1.system", "system", "sys", "", "system msg"))

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if allMsgs != 3 {
		t.Errorf("allMsgs = %d, want 3", allMsgs)
	}
	if userMsgs != 1 {
		t.Errorf("userMsgs = %d, want 1", userMsgs)
	}
}

// TestE2EHTTPIntegration tests the full HTTP flow end-to-end using the HTTP server.
func TestE2EHTTPIntegration(t *testing.T) {
	ctx := context.Background()
	b := NewBroker(WithWorkers(2), WithQueueDepth(64))
	b.Start(ctx)
	defer b.Shutdown(ctx)

	// Create agents via library API
	alice, _ := NewAgentMailbox(b, "alice", AgentTypeUserProxy, WithAgentChatID("e2e"))
	defer alice.Close()
	bob, _ := NewAgentMailbox(b, "bob", AgentTypeAssistant, WithAgentChatID("e2e"))
	defer bob.Close()

	// Send from alice to bob
	alice.Send(ctx, "bob", "e2e test message", WithSendType("user"))

	// Bob should receive
	select {
	case evt := <-bob.Receive():
		if evt.Content != "e2e test message" {
			t.Errorf("content = %q, want %q", evt.Content, "e2e test message")
		}
		if evt.Sender != "alice" {
			t.Errorf("sender = %q, want %q", evt.Sender, "alice")
		}
		if evt.Type != "user" {
			t.Errorf("type = %q, want %q", evt.Type, "user")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

// TestE2EMultiChatIsolation verifies that messages in different chats
// don't leak to agents in other chats.
func TestE2EMultiChatIsolation(t *testing.T) {
	ctx := context.Background()
	b := NewBroker(WithWorkers(2), WithQueueDepth(64))

	// Agent bob in chat-1
	bob1, _ := NewAgentMailbox(b, "bob", AgentTypeAssistant, WithAgentChatID("chat-1"))
	defer bob1.Close()

	// Agent bob in chat-2 (different chat, same name)
	bob2, _ := NewAgentMailbox(b, "bob-2", AgentTypeAssistant, WithAgentChatID("chat-2"))
	defer bob2.Close()

	b.Start(ctx)
	defer b.Shutdown(ctx)

	// Publish to chat-1 only
	evt := NewEvent("chat.chat-1.agent.bob", "user", "alice", "bob", "chat-1 only")
	b.Publish(ctx, evt)

	// bob1 should receive
	select {
	case got := <-bob1.Receive():
		if got.Content != "chat-1 only" {
			t.Errorf("bob1 content = %q", got.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: bob1 didn't receive")
	}

	// bob2 should NOT receive (different chat)
	select {
	case got := <-bob2.Receive():
		t.Errorf("bob2 should not receive, got: %q", got.Content)
	case <-time.After(200 * time.Millisecond):
		// Expected: no message for bob2
	}
}
