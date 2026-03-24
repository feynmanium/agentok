package edqlite

import (
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	evt := NewEvent("chat.1.agent.bob", "user", "alice", "bob", "hello")

	if evt.ID == "" {
		t.Error("expected non-empty ID")
	}
	if evt.Topic != "chat.1.agent.bob" {
		t.Errorf("topic = %q, want %q", evt.Topic, "chat.1.agent.bob")
	}
	if evt.Type != "user" {
		t.Errorf("type = %q, want %q", evt.Type, "user")
	}
	if evt.Sender != "alice" {
		t.Errorf("sender = %q, want %q", evt.Sender, "alice")
	}
	if evt.Receiver != "bob" {
		t.Errorf("receiver = %q, want %q", evt.Receiver, "bob")
	}
	if evt.Content != "hello" {
		t.Errorf("content = %q, want %q", evt.Content, "hello")
	}
	if evt.Priority != PriorityNormal {
		t.Errorf("priority = %d, want %d", evt.Priority, PriorityNormal)
	}
	if time.Since(evt.CreatedAt) > time.Second {
		t.Error("created_at should be recent")
	}
}

func TestEventWithPriority(t *testing.T) {
	evt := NewEvent("chat.1.broadcast", "system", "sys", "", "test").WithPriority(PriorityUrgent)
	if evt.Priority != PriorityUrgent {
		t.Errorf("priority = %d, want %d", evt.Priority, PriorityUrgent)
	}
}

func TestEventWithMetadata(t *testing.T) {
	meta := map[string]any{"key": "value"}
	evt := NewEvent("chat.1.broadcast", "system", "sys", "", "test").WithMetadata(meta)
	if evt.Metadata["key"] != "value" {
		t.Errorf("metadata[key] = %v, want %q", evt.Metadata["key"], "value")
	}
}

func TestEventValidate(t *testing.T) {
	tests := []struct {
		name    string
		evt     Event
		wantErr bool
	}{
		{
			name:    "valid event",
			evt:     NewEvent("chat.1.agent.bob", "user", "alice", "bob", "hello"),
			wantErr: false,
		},
		{
			name:    "missing ID",
			evt:     Event{Topic: "a.b", Type: "user", Sender: "alice", Content: "hi"},
			wantErr: true,
		},
		{
			name:    "missing topic",
			evt:     Event{ID: "x", Type: "user", Sender: "alice", Content: "hi"},
			wantErr: true,
		},
		{
			name:    "missing type",
			evt:     Event{ID: "x", Topic: "a.b", Sender: "alice", Content: "hi"},
			wantErr: true,
		},
		{
			name:    "missing sender",
			evt:     Event{ID: "x", Topic: "a.b", Type: "user", Content: "hi"},
			wantErr: true,
		},
		{
			name:    "missing content",
			evt:     Event{ID: "x", Topic: "a.b", Type: "user", Sender: "alice"},
			wantErr: true,
		},
		{
			name:    "invalid topic with wildcard",
			evt:     Event{ID: "x", Topic: "a.*", Type: "user", Sender: "alice", Content: "hi"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.evt.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPriorityString(t *testing.T) {
	tests := []struct {
		p    Priority
		want string
	}{
		{PriorityLow, "low"},
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityUrgent, "urgent"},
		{Priority(99), "priority(99)"},
	}
	for _, tt := range tests {
		if got := tt.p.String(); got != tt.want {
			t.Errorf("Priority(%d).String() = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestGenerateIDUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateID()
		if seen[id] {
			t.Fatalf("duplicate ID: %s", id)
		}
		seen[id] = true
	}
}
