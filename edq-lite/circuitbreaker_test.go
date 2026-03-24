package edqlite

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreakerClosedState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	if cb.State() != CircuitClosed {
		t.Errorf("initial state = %v, want closed", cb.State())
	}

	// Successful calls should keep it closed
	for i := 0; i < 10; i++ {
		err := cb.Execute(func() error { return nil })
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	}

	if cb.State() != CircuitClosed {
		t.Errorf("state after successes = %v, want closed", cb.State())
	}
}

func TestCircuitBreakerOpensAfterMaxFailures(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        5 * time.Second,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	fail := errors.New("fail")

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(func() error { return fail })
	}

	if cb.State() != CircuitOpen {
		t.Errorf("state after %d failures = %v, want open", cfg.MaxFailures, cb.State())
	}

	// Should block requests
	err := cb.Execute(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreakerHalfOpenRecovery(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	fail := errors.New("fail")

	// Trip it
	cb.Execute(func() error { return fail })
	cb.Execute(func() error { return fail })
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}

	// Wait for reset timeout
	time.Sleep(100 * time.Millisecond)

	// Should transition to half-open and allow one request
	if cb.State() != CircuitHalfOpen {
		t.Fatalf("expected half-open after timeout, got %v", cb.State())
	}

	// Successful probe should close the circuit
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("probe Execute() error = %v", err)
	}

	if cb.State() != CircuitClosed {
		t.Errorf("state after successful probe = %v, want closed", cb.State())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	fail := errors.New("fail")

	// Trip it
	cb.Execute(func() error { return fail })
	cb.Execute(func() error { return fail })

	// Wait for reset timeout
	time.Sleep(100 * time.Millisecond)

	// Failed probe should go back to open
	cb.Execute(func() error { return fail })

	if cb.State() != CircuitOpen {
		t.Errorf("state after failed probe = %v, want open", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        5 * time.Second,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	fail := errors.New("fail")
	cb.Execute(func() error { return fail })
	cb.Execute(func() error { return fail })

	if cb.State() != CircuitOpen {
		t.Fatalf("expected open")
	}

	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("state after Reset = %v, want closed", cb.State())
	}

	// Should accept requests again
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("Execute after Reset error = %v", err)
	}
}

func TestCircuitBreakerSuccessResetsFailureCount(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        5 * time.Second,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	fail := errors.New("fail")

	// 2 failures, then a success
	cb.Execute(func() error { return fail })
	cb.Execute(func() error { return fail })
	cb.Execute(func() error { return nil }) // resets counter

	// 2 more failures should not trip (counter was reset)
	cb.Execute(func() error { return fail })
	cb.Execute(func() error { return fail })

	if cb.State() != CircuitClosed {
		t.Errorf("state = %v, want closed (success should have reset counter)", cb.State())
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	fail := errors.New("fail")

	cb.Execute(func() error { return nil })  // success
	cb.Execute(func() error { return fail }) // failure
	cb.Execute(func() error { return fail }) // failure -> trips

	// Try while open
	cb.Execute(func() error { return nil }) // blocked

	stats := cb.Stats()
	if stats.TotalSuccesses != 1 {
		t.Errorf("total_successes = %d, want 1", stats.TotalSuccesses)
	}
	if stats.TotalFailures != 2 {
		t.Errorf("total_failures = %d, want 2", stats.TotalFailures)
	}
	if stats.TotalTripped != 1 {
		t.Errorf("total_tripped = %d, want 1", stats.TotalTripped)
	}
	if stats.TotalBlocked != 1 {
		t.Errorf("total_blocked = %d, want 1", stats.TotalBlocked)
	}
	if stats.State != "open" {
		t.Errorf("state = %q, want %q", stats.State, "open")
	}
}

func TestCircuitBreakerOnStateChange(t *testing.T) {
	transitions := make([]string, 0)
	var mu sync.Mutex

	cfg := CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxAttempts: 1,
		OnStateChange: func(from, to CircuitState) {
			mu.Lock()
			transitions = append(transitions, from.String()+"->"+to.String())
			mu.Unlock()
		},
	}
	cb := NewCircuitBreaker(cfg)

	fail := errors.New("fail")
	cb.Execute(func() error { return fail })
	cb.Execute(func() error { return fail }) // trips

	// Wait for callback goroutine
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(transitions) != 1 || transitions[0] != "closed->open" {
		t.Errorf("transitions = %v, want [closed->open]", transitions)
	}
	mu.Unlock()

	// Wait for timeout, trigger half-open
	time.Sleep(100 * time.Millisecond)
	cb.State()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(transitions) < 2 {
		t.Errorf("expected at least 2 transitions, got %v", transitions)
	}
	mu.Unlock()
}

func TestCircuitBreakerHalfOpenMaxAttempts(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxFailures:         1,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxAttempts: 2,
	}
	cb := NewCircuitBreaker(cfg)

	fail := errors.New("fail")
	cb.Execute(func() error { return fail }) // trips

	time.Sleep(100 * time.Millisecond)

	// Should allow 2 half-open attempts
	var allowed int
	for i := 0; i < 5; i++ {
		err := cb.Execute(func() error { return nil })
		if err == nil {
			allowed++
		}
	}

	// After 2 successes the circuit closes, so subsequent calls also succeed
	if allowed < 2 {
		t.Errorf("allowed %d requests in half-open, want >= 2", allowed)
	}
}

func TestCircuitBreakerConcurrent(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxFailures:         10,
		ResetTimeout:        100 * time.Millisecond,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			cb.Execute(func() error { return nil })
		}()
	}
	wg.Wait()

	stats := cb.Stats()
	if stats.TotalSuccesses != int64(n) {
		t.Errorf("total_successes = %d, want %d", stats.TotalSuccesses, n)
	}
}

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "state(99)"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestProtectedHandler(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        5 * time.Second,
		HalfOpenMaxAttempts: 1,
	})

	var called atomic.Int32
	handler := ProtectedHandler(cb, func(evt Event) error {
		called.Add(1)
		return errors.New("always fails")
	})

	evt := NewEvent("t.a", "user", "alice", "bob", "hi")

	// First two calls go through (and fail)
	handler(evt)
	handler(evt)

	// Circuit should be open now, third call blocked
	err := handler(evt)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}

	if called.Load() != 2 {
		t.Errorf("handler called %d times, want 2", called.Load())
	}
}

func TestProtectedSubscription(t *testing.T) {
	b := NewBroker(WithWorkers(1), WithQueueDepth(16))
	ctx := context.Background()

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        5 * time.Second,
		HalfOpenMaxAttempts: 1,
	})

	var received atomic.Int32
	subID, err := ProtectedSubscription(b, "test.>", cb, func(evt Event) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("ProtectedSubscription error = %v", err)
	}

	b.Start(ctx)
	defer b.Shutdown(ctx)

	b.Publish(ctx, NewEvent("test.topic", "user", "alice", "bob", "hi"))
	time.Sleep(200 * time.Millisecond)

	if received.Load() != 1 {
		t.Errorf("received = %d, want 1", received.Load())
	}

	b.Unsubscribe(subID)
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	if cfg.MaxFailures != 5 {
		t.Errorf("MaxFailures = %d, want 5", cfg.MaxFailures)
	}
	if cfg.ResetTimeout != 30*time.Second {
		t.Errorf("ResetTimeout = %v, want 30s", cfg.ResetTimeout)
	}
	if cfg.HalfOpenMaxAttempts != 1 {
		t.Errorf("HalfOpenMaxAttempts = %d, want 1", cfg.HalfOpenMaxAttempts)
	}
}

func TestCircuitBreakerDefaultConfig(t *testing.T) {
	// Zero-value config should get defaults applied
	cb := NewCircuitBreaker(CircuitBreakerConfig{})
	if cb.cfg.MaxFailures != 5 {
		t.Errorf("MaxFailures = %d, want default 5", cb.cfg.MaxFailures)
	}
	if cb.cfg.ResetTimeout != 30*time.Second {
		t.Errorf("ResetTimeout = %v, want default 30s", cb.cfg.ResetTimeout)
	}
}
