package edqlite

import "testing"

func TestSubscriptionMatches(t *testing.T) {
	evt := NewEvent("chat.1.agent.bob", "user", "alice", "bob", "hello")

	tests := []struct {
		name    string
		sub     Subscription
		want    bool
	}{
		{
			name: "exact match no filter",
			sub: Subscription{
				Pattern: "chat.1.agent.bob",
				Handler: func(Event) error { return nil },
			},
			want: true,
		},
		{
			name: "wildcard match no filter",
			sub: Subscription{
				Pattern: "chat.1.agent.*",
				Handler: func(Event) error { return nil },
			},
			want: true,
		},
		{
			name: "no match",
			sub: Subscription{
				Pattern: "chat.2.agent.bob",
				Handler: func(Event) error { return nil },
			},
			want: false,
		},
		{
			name: "match but filter rejects",
			sub: Subscription{
				Pattern: "chat.1.agent.bob",
				Filter:  func(e Event) bool { return e.Type == "system" },
				Handler: func(Event) error { return nil },
			},
			want: false,
		},
		{
			name: "match and filter accepts",
			sub: Subscription{
				Pattern: "chat.1.agent.bob",
				Filter:  func(e Event) bool { return e.Type == "user" },
				Handler: func(Event) error { return nil },
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sub.matches(evt); got != tt.want {
				t.Errorf("matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
