package edqlite

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// RuleAction is the result of a rule evaluation on a single event.
type RuleAction int

const (
	// RulePass lets the event continue to the next rule in the chain.
	RulePass RuleAction = iota
	// RuleTransform indicates the event was modified and should continue.
	RuleTransform
	// RuleDrop silently discards the event.
	RuleDrop
	// RuleReject discards the event and records an error.
	RuleReject
	// RuleRoute overrides the event's topic and continues processing.
	RuleRoute
)

// String returns the human-readable name for a RuleAction.
func (a RuleAction) String() string {
	switch a {
	case RulePass:
		return "pass"
	case RuleTransform:
		return "transform"
	case RuleDrop:
		return "drop"
	case RuleReject:
		return "reject"
	case RuleRoute:
		return "route"
	default:
		return fmt.Sprintf("action(%d)", int(a))
	}
}

// RuleResult is the outcome of applying a rule to a single event record.
type RuleResult struct {
	Action  RuleAction
	Event   Event  // The (possibly modified) event
	Reason  string // Explanation for drop/reject
	RouteTo string // New topic for RuleRoute action
}

// Rule evaluates a single event and returns a result.
// Rules are applied record-by-record in a chain.
type Rule interface {
	// Name returns a unique identifier for this rule.
	Name() string
	// Evaluate processes a single event and returns the action to take.
	Evaluate(ctx context.Context, evt Event) RuleResult
}

// RuleFunc is a convenience adapter to use ordinary functions as Rules.
type RuleFunc struct {
	RuleName string
	Fn       func(ctx context.Context, evt Event) RuleResult
}

func (r *RuleFunc) Name() string {
	return r.RuleName
}

func (r *RuleFunc) Evaluate(ctx context.Context, evt Event) RuleResult {
	return r.Fn(ctx, evt)
}

// RuleChain processes events through an ordered list of rules, record by record.
// Processing stops at the first rule that returns Drop, Reject, or Route.
type RuleChain struct {
	rules []Rule
	mu    sync.RWMutex
	stats ruleChainStats
}

type ruleChainStats struct {
	processed int64
	passed    int64
	dropped   int64
	rejected  int64
	routed    int64
	mu        sync.Mutex
}

// RuleChainStats contains rule chain execution statistics.
type RuleChainStats struct {
	Processed int64 `json:"processed"`
	Passed    int64 `json:"passed"`
	Dropped   int64 `json:"dropped"`
	Rejected  int64 `json:"rejected"`
	Routed    int64 `json:"routed"`
}

// NewRuleChain creates an empty rule chain.
func NewRuleChain(rules ...Rule) *RuleChain {
	return &RuleChain{
		rules: rules,
	}
}

// AddRule appends a rule to the end of the chain.
func (rc *RuleChain) AddRule(r Rule) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.rules = append(rc.rules, r)
}

// InsertRule inserts a rule at the given position.
func (rc *RuleChain) InsertRule(pos int, r Rule) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if pos < 0 || pos > len(rc.rules) {
		return fmt.Errorf("rule chain: position %d out of range [0, %d]", pos, len(rc.rules))
	}
	rc.rules = append(rc.rules[:pos], append([]Rule{r}, rc.rules[pos:]...)...)
	return nil
}

// RemoveRule removes the first rule with the given name.
func (rc *RuleChain) RemoveRule(name string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for i, r := range rc.rules {
		if r.Name() == name {
			rc.rules = append(rc.rules[:i], rc.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule chain: rule %q not found", name)
}

// Rules returns a copy of the current rule list.
func (rc *RuleChain) Rules() []Rule {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	out := make([]Rule, len(rc.rules))
	copy(out, rc.rules)
	return out
}

// Process applies the chain to a single event record.
// Returns the final result after all rules have been applied.
func (rc *RuleChain) Process(ctx context.Context, evt Event) RuleResult {
	rc.mu.RLock()
	rules := make([]Rule, len(rc.rules))
	copy(rules, rc.rules)
	rc.mu.RUnlock()

	rc.stats.mu.Lock()
	rc.stats.processed++
	rc.stats.mu.Unlock()

	current := evt
	for _, rule := range rules {
		result := rule.Evaluate(ctx, current)
		switch result.Action {
		case RulePass:
			continue
		case RuleTransform:
			current = result.Event
		case RuleDrop:
			rc.stats.mu.Lock()
			rc.stats.dropped++
			rc.stats.mu.Unlock()
			return RuleResult{Action: RuleDrop, Event: current, Reason: result.Reason}
		case RuleReject:
			rc.stats.mu.Lock()
			rc.stats.rejected++
			rc.stats.mu.Unlock()
			return RuleResult{Action: RuleReject, Event: current, Reason: result.Reason}
		case RuleRoute:
			rc.stats.mu.Lock()
			rc.stats.routed++
			rc.stats.mu.Unlock()
			current.Topic = result.RouteTo
			return RuleResult{Action: RuleRoute, Event: current, RouteTo: result.RouteTo}
		}
	}

	rc.stats.mu.Lock()
	rc.stats.passed++
	rc.stats.mu.Unlock()
	return RuleResult{Action: RulePass, Event: current}
}

// ProcessBatch applies the chain to each event in the batch individually.
// Returns results in the same order as the input events.
func (rc *RuleChain) ProcessBatch(ctx context.Context, events []Event) []RuleResult {
	results := make([]RuleResult, len(events))
	for i, evt := range events {
		results[i] = rc.Process(ctx, evt)
	}
	return results
}

// Stats returns rule chain execution statistics.
func (rc *RuleChain) Stats() RuleChainStats {
	rc.stats.mu.Lock()
	defer rc.stats.mu.Unlock()
	return RuleChainStats{
		Processed: rc.stats.processed,
		Passed:    rc.stats.passed,
		Dropped:   rc.stats.dropped,
		Rejected:  rc.stats.rejected,
		Routed:    rc.stats.routed,
	}
}

// RuleProcessor integrates a RuleChain with the Broker.
// It subscribes to a topic pattern, processes each event through the chain,
// and republishes passing/routed events.
type RuleProcessor struct {
	broker *Broker
	chain  *RuleChain
	subID  string
	cancel context.CancelFunc
}

// NewRuleProcessor creates a rule processor that applies the chain to events
// matching the given pattern, then republishes them to their (possibly modified) topic.
func NewRuleProcessor(broker *Broker, pattern TopicPattern, chain *RuleChain) (*RuleProcessor, error) {
	if chain == nil {
		return nil, errors.New("rule processor: chain must not be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	rp := &RuleProcessor{
		broker: broker,
		chain:  chain,
		cancel: cancel,
	}

	subID, err := broker.Subscribe(pattern, func(evt Event) error {
		result := chain.Process(ctx, evt)
		switch result.Action {
		case RulePass, RuleTransform:
			// Event passes through — no re-publish needed since it's already dispatched
			return nil
		case RuleRoute:
			// Re-publish to the new topic
			return broker.Publish(ctx, result.Event)
		case RuleDrop:
			return nil
		case RuleReject:
			return fmt.Errorf("rejected: %s", result.Reason)
		}
		return nil
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("rule processor: subscribe failed: %w", err)
	}
	rp.subID = subID
	return rp, nil
}

// Close stops the rule processor.
func (rp *RuleProcessor) Close() error {
	rp.cancel()
	return rp.broker.Unsubscribe(rp.subID)
}

// Chain returns the underlying rule chain.
func (rp *RuleProcessor) Chain() *RuleChain {
	return rp.chain
}

// --- Built-in Rules ---

// FilterByTypeRule drops events that don't match any of the allowed types.
func FilterByTypeRule(allowedTypes ...string) Rule {
	allowed := make(map[string]bool, len(allowedTypes))
	for _, t := range allowedTypes {
		allowed[t] = true
	}
	return &RuleFunc{
		RuleName: "filter_by_type",
		Fn: func(_ context.Context, evt Event) RuleResult {
			if allowed[evt.Type] {
				return RuleResult{Action: RulePass}
			}
			return RuleResult{Action: RuleDrop, Reason: fmt.Sprintf("type %q not in allowed list", evt.Type)}
		},
	}
}

// FilterBySenderRule drops events not from one of the allowed senders.
func FilterBySenderRule(allowedSenders ...string) Rule {
	allowed := make(map[string]bool, len(allowedSenders))
	for _, s := range allowedSenders {
		allowed[s] = true
	}
	return &RuleFunc{
		RuleName: "filter_by_sender",
		Fn: func(_ context.Context, evt Event) RuleResult {
			if allowed[evt.Sender] {
				return RuleResult{Action: RulePass}
			}
			return RuleResult{Action: RuleDrop, Reason: fmt.Sprintf("sender %q not allowed", evt.Sender)}
		},
	}
}

// PriorityBoostRule upgrades priority for events matching a predicate.
func PriorityBoostRule(name string, pred func(Event) bool, newPriority Priority) Rule {
	return &RuleFunc{
		RuleName: name,
		Fn: func(_ context.Context, evt Event) RuleResult {
			if pred(evt) {
				evt.Priority = newPriority
				return RuleResult{Action: RuleTransform, Event: evt}
			}
			return RuleResult{Action: RulePass}
		},
	}
}

// ContentRewriteRule modifies event content using a transform function.
func ContentRewriteRule(name string, transform func(string) string) Rule {
	return &RuleFunc{
		RuleName: name,
		Fn: func(_ context.Context, evt Event) RuleResult {
			evt.Content = transform(evt.Content)
			return RuleResult{Action: RuleTransform, Event: evt}
		},
	}
}

// MetadataEnrichRule adds metadata fields to events.
func MetadataEnrichRule(name string, enrichments map[string]any) Rule {
	return &RuleFunc{
		RuleName: name,
		Fn: func(_ context.Context, evt Event) RuleResult {
			if evt.Metadata == nil {
				evt.Metadata = make(map[string]any)
			}
			for k, v := range enrichments {
				evt.Metadata[k] = v
			}
			return RuleResult{Action: RuleTransform, Event: evt}
		},
	}
}

// MaxContentLengthRule rejects events whose content exceeds the limit.
func MaxContentLengthRule(maxLen int) Rule {
	return &RuleFunc{
		RuleName: "max_content_length",
		Fn: func(_ context.Context, evt Event) RuleResult {
			if len(evt.Content) > maxLen {
				return RuleResult{
					Action: RuleReject,
					Reason: fmt.Sprintf("content length %d exceeds max %d", len(evt.Content), maxLen),
				}
			}
			return RuleResult{Action: RulePass}
		},
	}
}

// ConditionalRouteRule re-routes events matching a predicate to a new topic.
func ConditionalRouteRule(name string, pred func(Event) bool, targetTopic string) Rule {
	return &RuleFunc{
		RuleName: name,
		Fn: func(_ context.Context, evt Event) RuleResult {
			if pred(evt) {
				return RuleResult{Action: RuleRoute, RouteTo: targetTopic}
			}
			return RuleResult{Action: RulePass}
		},
	}
}
