package edqlite

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

// TopicInfo describes an active topic pattern and its subscriber count.
type TopicInfo struct {
	Pattern    string `json:"pattern"`
	Subscribers int   `json:"subscribers"`
}

// BrokerOption configures broker behavior.
type BrokerOption func(*brokerConfig)

type brokerConfig struct {
	queueDepth        int
	workers           int
	deadLetterHandler HandlerFunc
}

// WithQueueDepth sets the internal channel buffer size. Default: 1024.
func WithQueueDepth(n int) BrokerOption {
	return func(c *brokerConfig) { c.queueDepth = n }
}

// WithWorkers sets the number of dispatch worker goroutines. Default: runtime.NumCPU().
func WithWorkers(n int) BrokerOption {
	return func(c *brokerConfig) { c.workers = n }
}

// WithDeadLetterHandler sets a handler for events that fail delivery to all subscribers.
func WithDeadLetterHandler(h HandlerFunc) BrokerOption {
	return func(c *brokerConfig) { c.deadLetterHandler = h }
}

// SubscribeOption configures a subscription.
type SubscribeOption func(*subscribeConfig)

type subscribeConfig struct {
	filter Filter
	id     string
}

// WithFilter adds a filter predicate to a subscription.
func WithFilter(f Filter) SubscribeOption {
	return func(c *subscribeConfig) { c.filter = f }
}

// WithSubscriptionID sets a specific ID for the subscription.
func WithSubscriptionID(id string) SubscribeOption {
	return func(c *subscribeConfig) { c.id = id }
}

// Broker is the central event dispatch engine.
type Broker struct {
	cfg           brokerConfig
	subs          map[string]*Subscription
	mu            sync.RWMutex
	ingest        chan Event
	done          chan struct{}
	wg            sync.WaitGroup
	started       atomic.Bool
	stopped       atomic.Bool
	published     atomic.Int64
	delivered     atomic.Int64
	deadLettered  atomic.Int64
}

// NewBroker creates a new broker with the given options.
func NewBroker(opts ...BrokerOption) *Broker {
	cfg := brokerConfig{
		queueDepth: 1024,
		workers:    runtime.NumCPU(),
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.workers < 1 {
		cfg.workers = 1
	}

	return &Broker{
		cfg:  cfg,
		subs: make(map[string]*Subscription),
		done: make(chan struct{}),
	}
}

// Start begins background dispatch workers. Must be called before Publish.
func (b *Broker) Start(ctx context.Context) error {
	if b.started.Swap(true) {
		return errors.New("broker: already started")
	}
	b.ingest = make(chan Event, b.cfg.queueDepth)

	for i := 0; i < b.cfg.workers; i++ {
		b.wg.Add(1)
		go b.worker(ctx)
	}
	return nil
}

// Publish enqueues an event for dispatch. Thread-safe.
func (b *Broker) Publish(ctx context.Context, evt Event) error {
	if !b.started.Load() {
		return errors.New("broker: not started")
	}
	if b.stopped.Load() {
		return errors.New("broker: stopped")
	}
	if err := evt.Validate(); err != nil {
		return fmt.Errorf("broker: %w", err)
	}
	select {
	case b.ingest <- evt:
		b.published.Add(1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Subscribe registers a subscription and returns its ID.
func (b *Broker) Subscribe(pattern TopicPattern, handler HandlerFunc, opts ...SubscribeOption) (string, error) {
	if err := ValidatePattern(string(pattern)); err != nil {
		return "", fmt.Errorf("broker: %w", err)
	}
	if handler == nil {
		return "", errors.New("broker: handler must not be nil")
	}

	cfg := subscribeConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	id := cfg.id
	if id == "" {
		id = generateID()
	}

	sub := &Subscription{
		ID:      id,
		Pattern: pattern,
		Filter:  cfg.filter,
		Handler: handler,
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.subs[id]; exists {
		return "", fmt.Errorf("broker: subscription %q already exists", id)
	}
	b.subs[id] = sub
	return id, nil
}

// Unsubscribe removes a subscription by ID.
func (b *Broker) Unsubscribe(subID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.subs[subID]; !exists {
		return fmt.Errorf("broker: subscription %q not found", subID)
	}
	delete(b.subs, subID)
	return nil
}

// Shutdown gracefully drains the queue and stops workers.
func (b *Broker) Shutdown(ctx context.Context) error {
	if !b.started.Load() {
		return nil
	}
	b.stopped.Store(true)
	close(b.ingest)
	b.wg.Wait()
	close(b.done)
	return nil
}

// Topics returns active topic patterns with subscriber counts.
func (b *Broker) Topics() []TopicInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()

	counts := make(map[string]int)
	for _, sub := range b.subs {
		counts[string(sub.Pattern)]++
	}

	result := make([]TopicInfo, 0, len(counts))
	for pattern, count := range counts {
		result = append(result, TopicInfo{Pattern: pattern, Subscribers: count})
	}
	return result
}

// Stats returns broker statistics.
func (b *Broker) Stats() BrokerStats {
	return BrokerStats{
		Published:    b.published.Load(),
		Delivered:    b.delivered.Load(),
		DeadLettered: b.deadLettered.Load(),
	}
}

// BrokerStats contains broker statistics.
type BrokerStats struct {
	Published    int64 `json:"published"`
	Delivered    int64 `json:"delivered"`
	DeadLettered int64 `json:"dead_lettered"`
}

func (b *Broker) worker(ctx context.Context) {
	defer b.wg.Done()
	for evt := range b.ingest {
		b.dispatch(ctx, evt)
	}
}

func (b *Broker) dispatch(_ context.Context, evt Event) {
	b.mu.RLock()
	// Collect matching subscriptions
	matching := make([]*Subscription, 0, 4)
	for _, sub := range b.subs {
		if sub.matches(evt) {
			matching = append(matching, sub)
		}
	}
	b.mu.RUnlock()

	if len(matching) == 0 {
		b.deadLettered.Add(1)
		if b.cfg.deadLetterHandler != nil {
			_ = b.cfg.deadLetterHandler(evt)
		}
		return
	}

	for _, sub := range matching {
		if err := sub.Handler(evt); err != nil {
			b.deadLettered.Add(1)
			if b.cfg.deadLetterHandler != nil {
				_ = b.cfg.deadLetterHandler(evt)
			}
		} else {
			b.delivered.Add(1)
		}
	}
}
