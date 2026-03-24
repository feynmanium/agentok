package edqlite_test

import (
	"context"
	"fmt"
	"time"

	edqlite "github.com/dustland/agentok/edq-lite"
)

func Example_basicPubSub() {
	ctx := context.Background()
	broker := edqlite.NewBroker(edqlite.WithWorkers(1), edqlite.WithQueueDepth(16))

	done := make(chan struct{})
	broker.Subscribe("greetings.>", func(evt edqlite.Event) error {
		fmt.Printf("Received: %s from %s\n", evt.Content, evt.Sender)
		close(done)
		return nil
	})

	broker.Start(ctx)
	defer broker.Shutdown(ctx)

	evt := edqlite.NewEvent("greetings.hello", "user", "alice", "bob", "Hello, World!")
	broker.Publish(ctx, evt)

	select {
	case <-done:
	case <-time.After(time.Second):
	}
	// Output: Received: Hello, World! from alice
}

func Example_agentMailbox() {
	ctx := context.Background()
	broker := edqlite.NewBroker(edqlite.WithWorkers(2), edqlite.WithQueueDepth(32))

	alice, _ := edqlite.NewAgentMailbox(broker, "alice", edqlite.AgentTypeUserProxy, edqlite.WithAgentChatID("demo"))
	bob, _ := edqlite.NewAgentMailbox(broker, "bob", edqlite.AgentTypeAssistant, edqlite.WithAgentChatID("demo"))

	broker.Start(ctx)
	defer broker.Shutdown(ctx)
	defer alice.Close()
	defer bob.Close()

	alice.Send(ctx, "bob", "What is 2+2?", edqlite.WithSendType("user"))

	select {
	case evt := <-bob.Receive():
		fmt.Printf("%s asks: %s\n", evt.Sender, evt.Content)
	case <-time.After(time.Second):
		fmt.Println("timeout")
	}
	// Output: alice asks: What is 2+2?
}

func Example_topicRouter() {
	ctx := context.Background()
	broker := edqlite.NewBroker(edqlite.WithWorkers(1), edqlite.WithQueueDepth(16))
	router := edqlite.NewTopicRouter(broker)

	// Load edges from an Agentok flow graph
	edges := []edqlite.Edge{
		{Source: "user", Target: "assistant"},
		{Source: "assistant", Target: "reviewer"},
	}
	router.LoadFromEdges("session-1", edges)

	done := make(chan struct{})
	broker.Subscribe("chat.session-1.agent.assistant", func(evt edqlite.Event) error {
		fmt.Printf("Routed to assistant: %s\n", evt.Content)
		close(done)
		return nil
	})

	broker.Start(ctx)
	defer broker.Shutdown(ctx)

	evt := edqlite.NewEvent("", "user", "user", "assistant", "Route this message")
	router.RouteAndPublish(ctx, evt)

	select {
	case <-done:
	case <-time.After(time.Second):
	}
	// Output: Routed to assistant: Route this message
}
