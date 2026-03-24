package edqlite

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// Priority levels for event dispatch ordering.
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityNormal Priority = 5
	PriorityHigh   Priority = 10
	PriorityUrgent Priority = 15
)

// String returns the human-readable priority name.
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	default:
		return fmt.Sprintf("priority(%d)", int(p))
	}
}

// Event is the fundamental unit of communication in EDQ Lite.
type Event struct {
	ID        string         `json:"id"`
	Topic     string         `json:"topic"`
	Type      string         `json:"type"`               // "user", "assistant", "summary", "system", "tool_call", "tool_response"
	Sender    string         `json:"sender"`              // Agent name or ID
	Receiver  string         `json:"receiver,omitempty"`  // Agent name, ID, or "" for broadcast
	Content   string         `json:"content"`
	Priority  Priority       `json:"priority"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// NewEvent creates a new Event with a generated ID and current timestamp.
func NewEvent(topic, eventType, sender, receiver, content string) Event {
	return Event{
		ID:        generateID(),
		Topic:     topic,
		Type:      eventType,
		Sender:    sender,
		Receiver:  receiver,
		Content:   content,
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
	}
}

// WithPriority returns a copy of the event with the given priority.
func (e Event) WithPriority(p Priority) Event {
	e.Priority = p
	return e
}

// WithMetadata returns a copy of the event with the given metadata.
func (e Event) WithMetadata(m map[string]any) Event {
	e.Metadata = m
	return e
}

// Validate checks that the event has all required fields.
func (e Event) Validate() error {
	if e.ID == "" {
		return errors.New("event: id is required")
	}
	if e.Topic == "" {
		return errors.New("event: topic is required")
	}
	if e.Type == "" {
		return errors.New("event: type is required")
	}
	if e.Sender == "" {
		return errors.New("event: sender is required")
	}
	if e.Content == "" {
		return errors.New("event: content is required")
	}
	if err := ValidateTopic(e.Topic); err != nil {
		return fmt.Errorf("event: %w", err)
	}
	return nil
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ExportGenerateID is an exported wrapper for generating unique IDs.
func ExportGenerateID() string {
	return generateID()
}
