package edqlite

import (
	"context"
	"strings"
	"testing"
)

func TestRuleActionString(t *testing.T) {
	tests := []struct {
		action RuleAction
		want   string
	}{
		{RulePass, "pass"},
		{RuleTransform, "transform"},
		{RuleDrop, "drop"},
		{RuleReject, "reject"},
		{RuleRoute, "route"},
		{RuleAction(99), "action(99)"},
	}
	for _, tt := range tests {
		if got := tt.action.String(); got != tt.want {
			t.Errorf("RuleAction(%d).String() = %q, want %q", tt.action, got, tt.want)
		}
	}
}

func TestRuleChainPassThrough(t *testing.T) {
	chain := NewRuleChain()
	ctx := context.Background()
	evt := NewEvent("test.topic", "user", "alice", "bob", "hello")

	result := chain.Process(ctx, evt)
	if result.Action != RulePass {
		t.Errorf("empty chain should pass, got %v", result.Action)
	}
	if result.Event.Content != "hello" {
		t.Errorf("event content should be unchanged")
	}
}

func TestRuleChainFilterByType(t *testing.T) {
	chain := NewRuleChain(FilterByTypeRule("user", "assistant"))
	ctx := context.Background()

	// Allowed type
	evt := NewEvent("test.topic", "user", "alice", "bob", "hi")
	result := chain.Process(ctx, evt)
	if result.Action != RulePass {
		t.Errorf("user type should pass, got %v", result.Action)
	}

	// Disallowed type
	evt2 := NewEvent("test.topic", "system", "sys", "", "ping")
	result2 := chain.Process(ctx, evt2)
	if result2.Action != RuleDrop {
		t.Errorf("system type should be dropped, got %v", result2.Action)
	}
}

func TestRuleChainFilterBySender(t *testing.T) {
	chain := NewRuleChain(FilterBySenderRule("alice", "bob"))
	ctx := context.Background()

	evt := NewEvent("test.topic", "user", "alice", "bob", "hi")
	if result := chain.Process(ctx, evt); result.Action != RulePass {
		t.Errorf("alice should pass, got %v", result.Action)
	}

	evt2 := NewEvent("test.topic", "user", "eve", "bob", "hi")
	if result := chain.Process(ctx, evt2); result.Action != RuleDrop {
		t.Errorf("eve should be dropped, got %v", result.Action)
	}
}

func TestRuleChainMaxContentLength(t *testing.T) {
	chain := NewRuleChain(MaxContentLengthRule(10))
	ctx := context.Background()

	short := NewEvent("test.topic", "user", "alice", "bob", "hi")
	if result := chain.Process(ctx, short); result.Action != RulePass {
		t.Errorf("short content should pass, got %v", result.Action)
	}

	long := NewEvent("test.topic", "user", "alice", "bob", "this is way too long")
	result := chain.Process(ctx, long)
	if result.Action != RuleReject {
		t.Errorf("long content should be rejected, got %v", result.Action)
	}
	if result.Reason == "" {
		t.Error("reject should include reason")
	}
}

func TestRuleChainPriorityBoost(t *testing.T) {
	chain := NewRuleChain(PriorityBoostRule("boost-urgent", func(evt Event) bool {
		return evt.Type == "tool_call"
	}, PriorityUrgent))
	ctx := context.Background()

	// Should be boosted
	evt := NewEvent("test.topic", "tool_call", "alice", "bob", "call")
	result := chain.Process(ctx, evt)
	if result.Action != RulePass {
		t.Errorf("should pass after transform, got %v", result.Action)
	}
	if result.Event.Priority != PriorityUrgent {
		t.Errorf("priority should be urgent, got %v", result.Event.Priority)
	}

	// Should not be boosted
	evt2 := NewEvent("test.topic", "user", "alice", "bob", "hi")
	result2 := chain.Process(ctx, evt2)
	if result2.Event.Priority != PriorityNormal {
		t.Errorf("priority should remain normal, got %v", result2.Event.Priority)
	}
}

func TestRuleChainContentRewrite(t *testing.T) {
	chain := NewRuleChain(ContentRewriteRule("uppercase", strings.ToUpper))
	ctx := context.Background()

	evt := NewEvent("test.topic", "user", "alice", "bob", "hello")
	result := chain.Process(ctx, evt)
	if result.Event.Content != "HELLO" {
		t.Errorf("content should be uppercased, got %q", result.Event.Content)
	}
}

func TestRuleChainMetadataEnrich(t *testing.T) {
	enrichments := map[string]any{"env": "production", "version": "1.0"}
	chain := NewRuleChain(MetadataEnrichRule("add-env", enrichments))
	ctx := context.Background()

	evt := NewEvent("test.topic", "user", "alice", "bob", "hello")
	result := chain.Process(ctx, evt)
	if result.Event.Metadata["env"] != "production" {
		t.Errorf("metadata[env] = %v, want %q", result.Event.Metadata["env"], "production")
	}
	if result.Event.Metadata["version"] != "1.0" {
		t.Errorf("metadata[version] = %v, want %q", result.Event.Metadata["version"], "1.0")
	}
}

func TestRuleChainConditionalRoute(t *testing.T) {
	chain := NewRuleChain(ConditionalRouteRule("route-system", func(evt Event) bool {
		return evt.Type == "system"
	}, "chat.default.system"))
	ctx := context.Background()

	// Should be routed
	evt := NewEvent("test.topic", "system", "sys", "", "alert")
	result := chain.Process(ctx, evt)
	if result.Action != RuleRoute {
		t.Errorf("system events should be routed, got %v", result.Action)
	}
	if result.RouteTo != "chat.default.system" {
		t.Errorf("route_to = %q, want %q", result.RouteTo, "chat.default.system")
	}

	// Should pass through
	evt2 := NewEvent("test.topic", "user", "alice", "bob", "hi")
	result2 := chain.Process(ctx, evt2)
	if result2.Action != RulePass {
		t.Errorf("user events should pass, got %v", result2.Action)
	}
}

func TestRuleChainMultipleRules(t *testing.T) {
	chain := NewRuleChain(
		FilterBySenderRule("alice", "bob"),
		FilterByTypeRule("user", "assistant"),
		MaxContentLengthRule(100),
		ContentRewriteRule("trim", strings.TrimSpace),
		MetadataEnrichRule("tag", map[string]any{"processed": true}),
	)
	ctx := context.Background()

	// Passes all rules
	evt := NewEvent("test.topic", "user", "alice", "bob", "  hello  ")
	result := chain.Process(ctx, evt)
	if result.Action != RulePass {
		t.Errorf("should pass all rules, got %v", result.Action)
	}
	if result.Event.Content != "hello" {
		t.Errorf("content should be trimmed, got %q", result.Event.Content)
	}
	if result.Event.Metadata["processed"] != true {
		t.Error("metadata should be enriched")
	}

	// Fails sender filter
	evt2 := NewEvent("test.topic", "user", "eve", "bob", "hi")
	result2 := chain.Process(ctx, evt2)
	if result2.Action != RuleDrop {
		t.Errorf("eve should be dropped, got %v", result2.Action)
	}

	// Fails type filter
	evt3 := NewEvent("test.topic", "system", "alice", "", "ping")
	result3 := chain.Process(ctx, evt3)
	if result3.Action != RuleDrop {
		t.Errorf("system type should be dropped, got %v", result3.Action)
	}
}

func TestRuleChainAddRemoveRules(t *testing.T) {
	chain := NewRuleChain()
	chain.AddRule(FilterByTypeRule("user"))

	if len(chain.Rules()) != 1 {
		t.Errorf("rules count = %d, want 1", len(chain.Rules()))
	}

	chain.AddRule(MaxContentLengthRule(50))
	if len(chain.Rules()) != 2 {
		t.Errorf("rules count = %d, want 2", len(chain.Rules()))
	}

	// Remove by name
	if err := chain.RemoveRule("filter_by_type"); err != nil {
		t.Errorf("RemoveRule() error = %v", err)
	}
	if len(chain.Rules()) != 1 {
		t.Errorf("rules count = %d after remove, want 1", len(chain.Rules()))
	}

	// Remove non-existent
	if err := chain.RemoveRule("nonexistent"); err == nil {
		t.Error("expected error removing non-existent rule")
	}
}

func TestRuleChainInsertRule(t *testing.T) {
	chain := NewRuleChain(
		FilterByTypeRule("user"),
		MaxContentLengthRule(100),
	)

	// Insert in the middle
	err := chain.InsertRule(1, ContentRewriteRule("upper", strings.ToUpper))
	if err != nil {
		t.Fatalf("InsertRule() error = %v", err)
	}

	rules := chain.Rules()
	if len(rules) != 3 {
		t.Fatalf("rules count = %d, want 3", len(rules))
	}
	if rules[1].Name() != "upper" {
		t.Errorf("rule[1] name = %q, want %q", rules[1].Name(), "upper")
	}

	// Out of range
	if err := chain.InsertRule(99, FilterByTypeRule("x")); err == nil {
		t.Error("expected error for out-of-range position")
	}
	if err := chain.InsertRule(-1, FilterByTypeRule("x")); err == nil {
		t.Error("expected error for negative position")
	}
}

func TestRuleChainProcessBatch(t *testing.T) {
	chain := NewRuleChain(FilterByTypeRule("user"))
	ctx := context.Background()

	events := []Event{
		NewEvent("t.a", "user", "alice", "bob", "msg1"),
		NewEvent("t.b", "system", "sys", "", "ping"),
		NewEvent("t.c", "user", "bob", "alice", "msg2"),
	}

	results := chain.ProcessBatch(ctx, events)
	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}
	if results[0].Action != RulePass {
		t.Errorf("results[0] should pass, got %v", results[0].Action)
	}
	if results[1].Action != RuleDrop {
		t.Errorf("results[1] should drop, got %v", results[1].Action)
	}
	if results[2].Action != RulePass {
		t.Errorf("results[2] should pass, got %v", results[2].Action)
	}
}

func TestRuleChainStats(t *testing.T) {
	chain := NewRuleChain(
		FilterByTypeRule("user"),
		MaxContentLengthRule(5),
	)
	ctx := context.Background()

	chain.Process(ctx, NewEvent("t.a", "user", "alice", "bob", "hi"))     // pass
	chain.Process(ctx, NewEvent("t.a", "system", "sys", "", "ping"))       // dropped by type
	chain.Process(ctx, NewEvent("t.a", "user", "alice", "bob", "too long content")) // rejected by length

	stats := chain.Stats()
	if stats.Processed != 3 {
		t.Errorf("processed = %d, want 3", stats.Processed)
	}
	if stats.Passed != 1 {
		t.Errorf("passed = %d, want 1", stats.Passed)
	}
	if stats.Dropped != 1 {
		t.Errorf("dropped = %d, want 1", stats.Dropped)
	}
	if stats.Rejected != 1 {
		t.Errorf("rejected = %d, want 1", stats.Rejected)
	}
}

func TestRuleChainShortCircuit(t *testing.T) {
	callOrder := make([]string, 0)
	chain := NewRuleChain(
		&RuleFunc{RuleName: "r1", Fn: func(_ context.Context, evt Event) RuleResult {
			callOrder = append(callOrder, "r1")
			return RuleResult{Action: RuleDrop, Reason: "blocked"}
		}},
		&RuleFunc{RuleName: "r2", Fn: func(_ context.Context, evt Event) RuleResult {
			callOrder = append(callOrder, "r2")
			return RuleResult{Action: RulePass}
		}},
	)
	ctx := context.Background()

	chain.Process(ctx, NewEvent("t.a", "user", "alice", "bob", "hi"))

	if len(callOrder) != 1 || callOrder[0] != "r1" {
		t.Errorf("only r1 should be called, got %v", callOrder)
	}
}

func TestRuleFuncName(t *testing.T) {
	r := &RuleFunc{RuleName: "test-rule"}
	if r.Name() != "test-rule" {
		t.Errorf("Name() = %q, want %q", r.Name(), "test-rule")
	}
}

func TestRuleProcessorWithBroker(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(32))
	ctx := context.Background()

	chain := NewRuleChain(FilterByTypeRule("user"))

	rp, err := NewRuleProcessor(b, "chat.1.>", chain)
	if err != nil {
		t.Fatalf("NewRuleProcessor error = %v", err)
	}
	defer rp.Close()

	if rp.Chain() != chain {
		t.Error("Chain() should return the chain")
	}

	b.Start(ctx)
	defer b.Shutdown(ctx)

	// The processor is subscribed and filtering — we just verify no panic
	b.Publish(ctx, NewEvent("chat.1.agent.bob", "user", "alice", "bob", "hi"))
	b.Publish(ctx, NewEvent("chat.1.agent.bob", "system", "sys", "bob", "ignored"))
}

func TestRuleProcessorNilChain(t *testing.T) {
	b := NewBroker()
	_, err := NewRuleProcessor(b, "test.>", nil)
	if err == nil {
		t.Error("expected error for nil chain")
	}
}

func TestFilterByTypeRuleNames(t *testing.T) {
	r := FilterByTypeRule("user")
	if r.Name() != "filter_by_type" {
		t.Errorf("Name() = %q, want %q", r.Name(), "filter_by_type")
	}
}

func TestFilterBySenderRuleNames(t *testing.T) {
	r := FilterBySenderRule("alice")
	if r.Name() != "filter_by_sender" {
		t.Errorf("Name() = %q, want %q", r.Name(), "filter_by_sender")
	}
}

func TestMaxContentLengthRuleName(t *testing.T) {
	r := MaxContentLengthRule(10)
	if r.Name() != "max_content_length" {
		t.Errorf("Name() = %q, want %q", r.Name(), "max_content_length")
	}
}

func TestMetadataEnrichPreservesExisting(t *testing.T) {
	chain := NewRuleChain(MetadataEnrichRule("enrich", map[string]any{"new_key": "new_val"}))
	ctx := context.Background()

	evt := NewEvent("t.a", "user", "alice", "bob", "hi").WithMetadata(map[string]any{"existing": "val"})
	result := chain.Process(ctx, evt)
	if result.Event.Metadata["existing"] != "val" {
		t.Error("existing metadata should be preserved")
	}
	if result.Event.Metadata["new_key"] != "new_val" {
		t.Error("new metadata should be added")
	}
}

func TestRuleChainRouteStats(t *testing.T) {
	chain := NewRuleChain(ConditionalRouteRule("reroute", func(Event) bool { return true }, "other.topic"))
	ctx := context.Background()

	chain.Process(ctx, NewEvent("t.a", "user", "alice", "bob", "hi"))

	stats := chain.Stats()
	if stats.Routed != 1 {
		t.Errorf("routed = %d, want 1", stats.Routed)
	}
}
