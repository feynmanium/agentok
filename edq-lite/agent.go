package edqlite

import (
	"context"
	"fmt"
	"sync"
)

// AgentType mirrors Agentok's node class_type values.
type AgentType string

const (
	AgentTypeConversable AgentType = "ConversableAgent"
	AgentTypeAssistant   AgentType = "AssistantAgent"
	AgentTypeUserProxy   AgentType = "UserProxyAgent"
	AgentTypeGroupChat   AgentType = "GroupChat"
	AgentTypeCaptain     AgentType = "CaptainAgent"
	AgentTypeWebSurfer   AgentType = "WebSurferAgent"
)

// AgentMailbox provides send/receive semantics for a single agent.
type AgentMailbox interface {
	// Name returns the agent's identifier.
	Name() string

	// Type returns the agent's type.
	Type() AgentType

	// Send publishes a message from this agent to a target (or broadcast if target is "").
	Send(ctx context.Context, target string, content string, opts ...SendOption) error

	// Receive returns a channel that delivers events addressed to this agent.
	Receive() <-chan Event

	// Close unsubscribes and releases resources.
	Close() error
}

// SendOption configures a send operation.
type SendOption func(*sendConfig)

type sendConfig struct {
	priority  Priority
	metadata  map[string]any
	eventType string
	chatID    string
}

// WithSendPriority sets the event priority.
func WithSendPriority(p Priority) SendOption {
	return func(c *sendConfig) { c.priority = p }
}

// WithSendMetadata sets the event metadata.
func WithSendMetadata(m map[string]any) SendOption {
	return func(c *sendConfig) { c.metadata = m }
}

// WithSendType sets the event type (e.g., "user", "assistant").
func WithSendType(t string) SendOption {
	return func(c *sendConfig) { c.eventType = t }
}

// WithChatID sets the chat ID for topic construction.
func WithChatID(id string) SendOption {
	return func(c *sendConfig) { c.chatID = id }
}

// AgentOption configures an agent mailbox.
type AgentOption func(*agentConfig)

type agentConfig struct {
	chatID     string
	bufferSize int
}

// WithAgentChatID sets the chat scope for the agent's subscriptions.
func WithAgentChatID(id string) AgentOption {
	return func(c *agentConfig) { c.chatID = id }
}

// WithBufferSize sets the receive channel buffer size.
func WithBufferSize(n int) AgentOption {
	return func(c *agentConfig) { c.bufferSize = n }
}

// agentMailbox is the default AgentMailbox implementation.
type agentMailbox struct {
	name      string
	agentType AgentType
	broker    *Broker
	chatID    string
	ch        chan Event
	subIDs    []string
	mu        sync.Mutex
	closed    bool
}

// NewAgentMailbox creates a mailbox connected to the broker.
func NewAgentMailbox(broker *Broker, name string, agentType AgentType, opts ...AgentOption) (AgentMailbox, error) {
	cfg := agentConfig{
		bufferSize: 64,
	}
	for _, o := range opts {
		o(&cfg)
	}

	m := &agentMailbox{
		name:      name,
		agentType: agentType,
		broker:    broker,
		chatID:    cfg.chatID,
		ch:        make(chan Event, cfg.bufferSize),
	}

	// Subscribe to direct messages
	handler := func(evt Event) error {
		select {
		case m.ch <- evt:
			return nil
		default:
			return fmt.Errorf("agent %s: receive buffer full", name)
		}
	}

	var patterns []TopicPattern
	if cfg.chatID != "" {
		// Subscribe to direct + broadcast topics for this chat
		patterns = append(patterns,
			TopicPattern(fmt.Sprintf("chat.%s.agent.%s", cfg.chatID, name)),
			TopicPattern(fmt.Sprintf("chat.%s.broadcast", cfg.chatID)),
		)
	} else {
		// Subscribe to all messages for this agent across chats
		patterns = append(patterns, TopicPattern(fmt.Sprintf("agent.%s.>", name)))
	}

	for _, p := range patterns {
		subID, err := broker.Subscribe(p, handler)
		if err != nil {
			// Cleanup already-created subscriptions
			for _, id := range m.subIDs {
				_ = broker.Unsubscribe(id)
			}
			return nil, fmt.Errorf("agent %s: subscribe failed: %w", name, err)
		}
		m.subIDs = append(m.subIDs, subID)
	}

	return m, nil
}

func (m *agentMailbox) Name() string      { return m.name }
func (m *agentMailbox) Type() AgentType   { return m.agentType }
func (m *agentMailbox) Receive() <-chan Event { return m.ch }

func (m *agentMailbox) Send(ctx context.Context, target string, content string, opts ...SendOption) error {
	cfg := sendConfig{
		priority:  PriorityNormal,
		eventType: string(m.agentType),
		chatID:    m.chatID,
	}
	for _, o := range opts {
		o(&cfg)
	}

	var topic string
	chatID := cfg.chatID
	if chatID == "" {
		chatID = "default"
	}
	if target == "" {
		topic = fmt.Sprintf("chat.%s.broadcast", chatID)
	} else {
		topic = fmt.Sprintf("chat.%s.agent.%s", chatID, target)
	}

	evt := NewEvent(topic, cfg.eventType, m.name, target, content)
	evt.Priority = cfg.priority
	evt.Metadata = cfg.metadata

	return m.broker.Publish(ctx, evt)
}

func (m *agentMailbox) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	for _, id := range m.subIDs {
		_ = m.broker.Unsubscribe(id)
	}
	close(m.ch)
	return nil
}
