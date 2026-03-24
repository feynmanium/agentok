package edqlite

import (
	"context"
	"fmt"
	"sync"
)

// TopicRouter provides rule-based routing for events.
// It maps agent-to-agent communication edges to topics.
type TopicRouter struct {
	broker         *Broker
	directRoutes   map[routeKey]string
	broadcastRoutes map[string]string
	mu             sync.RWMutex
}

type routeKey struct {
	source string
	target string
}

// NewTopicRouter creates a new router connected to the broker.
func NewTopicRouter(broker *Broker) *TopicRouter {
	return &TopicRouter{
		broker:          broker,
		directRoutes:    make(map[routeKey]string),
		broadcastRoutes: make(map[string]string),
	}
}

// AddRoute registers a routing rule: events from source to target go to the given topic.
func (r *TopicRouter) AddRoute(source, target, topic string) error {
	if err := ValidateTopic(topic); err != nil {
		return fmt.Errorf("router: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.directRoutes[routeKey{source: source, target: target}] = topic
	return nil
}

// AddBroadcastRoute routes all events from source to a broadcast topic.
func (r *TopicRouter) AddBroadcastRoute(source, broadcastTopic string) error {
	if err := ValidateTopic(broadcastTopic); err != nil {
		return fmt.Errorf("router: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.broadcastRoutes[source] = broadcastTopic
	return nil
}

// RouteEvent determines the topic for an event based on registered rules.
// If no matching route is found, it returns a default topic based on the chat ID.
func (r *TopicRouter) RouteEvent(evt Event) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try direct route first
	if topic, ok := r.directRoutes[routeKey{source: evt.Sender, target: evt.Receiver}]; ok {
		return topic
	}
	// Try broadcast route
	if topic, ok := r.broadcastRoutes[evt.Sender]; ok {
		return topic
	}
	// Default: construct from receiver
	if evt.Receiver != "" {
		return fmt.Sprintf("chat.default.agent.%s", evt.Receiver)
	}
	return "chat.default.broadcast"
}

// RouteAndPublish routes an event and publishes it to the determined topic.
func (r *TopicRouter) RouteAndPublish(ctx context.Context, evt Event) error {
	evt.Topic = r.RouteEvent(evt)
	return r.broker.Publish(ctx, evt)
}

// LoadFromEdges loads routing rules from an Agentok-style edge list.
// Each edge has a source and target agent name. The chatID scopes the topics.
func (r *TopicRouter) LoadFromEdges(chatID string, edges []Edge) error {
	for _, edge := range edges {
		topic := fmt.Sprintf("chat.%s.agent.%s", chatID, edge.Target)
		if err := r.AddRoute(edge.Source, edge.Target, topic); err != nil {
			return err
		}
		// Also add reverse route for bidirectional conversation
		reverseTopic := fmt.Sprintf("chat.%s.agent.%s", chatID, edge.Source)
		if err := r.AddRoute(edge.Target, edge.Source, reverseTopic); err != nil {
			return err
		}
	}
	return nil
}

// Edge represents a conversation edge in an Agentok flow graph.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}
