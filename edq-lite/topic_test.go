package edqlite

import "testing"

func TestTopicPatternMatch(t *testing.T) {
	tests := []struct {
		pattern TopicPattern
		topic   string
		want    bool
	}{
		// Exact match
		{"chat.1.agent.bob", "chat.1.agent.bob", true},
		{"chat.1.agent.bob", "chat.1.agent.alice", false},
		{"chat.1.agent.bob", "chat.2.agent.bob", false},

		// Single wildcard
		{"chat.1.agent.*", "chat.1.agent.bob", true},
		{"chat.1.agent.*", "chat.1.agent.alice", true},
		{"chat.1.agent.*", "chat.2.agent.bob", false},
		{"chat.*.agent.bob", "chat.1.agent.bob", true},
		{"chat.*.agent.bob", "chat.99.agent.bob", true},
		{"*.*.*.*", "a.b.c.d", true},
		{"*.*.*.*", "a.b.c", false},

		// Multi wildcard
		{"chat.1.>", "chat.1.agent.bob", true},
		{"chat.1.>", "chat.1.broadcast", true},
		{"chat.1.>", "chat.1.a.b.c", true},
		{"chat.1.>", "chat.1", false}, // ">" must match at least one segment
		{"chat.>", "chat.1.agent.bob", true},
		{">", "anything.at.all", true},
		{">", "single", true},

		// Edge cases
		{"a", "a", true},
		{"a", "b", false},
		{"a.b", "a", false},
		{"a", "a.b", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.pattern)+"->"+tt.topic, func(t *testing.T) {
			if got := tt.pattern.Match(tt.topic); got != tt.want {
				t.Errorf("TopicPattern(%q).Match(%q) = %v, want %v", tt.pattern, tt.topic, got, tt.want)
			}
		})
	}
}

func TestValidateTopic(t *testing.T) {
	tests := []struct {
		topic   string
		wantErr bool
	}{
		{"chat.1.agent.bob", false},
		{"simple", false},
		{"a.b.c.d.e", false},
		{"", true},
		{"a..b", true},
		{".a", true},
		{"a.", true},
		{"a.*.b", true},
		{"a.>", true},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			err := ValidateTopic(tt.topic)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTopic(%q) error = %v, wantErr %v", tt.topic, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePattern(t *testing.T) {
	tests := []struct {
		pattern string
		wantErr bool
	}{
		{"chat.1.agent.bob", false},
		{"chat.1.agent.*", false},
		{"chat.1.>", false},
		{">", false},
		{"*.*.*", false},
		{"", true},
		{"a..b", true},
		{"a.>.b", true}, // > not last
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			err := ValidatePattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePattern(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}
		})
	}
}
