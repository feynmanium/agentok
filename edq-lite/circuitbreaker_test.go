package edqlite

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_StartsClosedAndAllows(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{})
	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Error("expected Allow() to return true for a new circuit breaker")
	}
}

func TestCircuitBreaker_TripsOpenAfterMaxFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: time.Minute,
	})

	// Record 4 failures — should stay closed.
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if cb.State() != StateClosed {
		t.Errorf("expected closed after 4 failures, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Error("expected Allow() true while closed")
	}

	// 5th failure trips the circuit open.
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("expected open after 5 failures, got %s", cb.State())
	}
	if cb.Allow() {
		t.Error("expected Allow() false when circuit is open")
	}
}

func TestCircuitBreaker_ResetTimeoutTransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 10 * time.Millisecond,
	})

	// Trip the circuit open.
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	// Wait for the reset timeout to elapse.
	time.Sleep(15 * time.Millisecond)

	// Allow() should transition to half-open and return true.
	if !cb.Allow() {
		t.Error("expected Allow() true after reset timeout")
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("expected half-open, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenSuccessClosesCircuit(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:         5,
		ResetTimeout:        10 * time.Millisecond,
		HalfOpenMaxAttempts: 3,
	})

	// Trip open and wait for half-open.
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	time.Sleep(15 * time.Millisecond)
	cb.Allow() // triggers transition to half-open

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected half-open, got %s", cb.State())
	}

	// Record enough successes to close the circuit.
	for i := 0; i < 3; i++ {
		cb.RecordSuccess()
	}

	if cb.State() != StateClosed {
		t.Errorf("expected closed after %d successes in half-open, got %s", 3, cb.State())
	}
	if !cb.Allow() {
		t.Error("expected Allow() true after circuit closed")
	}
}

func TestCircuitBreaker_HalfOpenFailureTripsOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:         5,
		ResetTimeout:        10 * time.Millisecond,
		HalfOpenMaxAttempts: 3,
	})

	// Trip open and wait for half-open.
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	time.Sleep(15 * time.Millisecond)
	cb.Allow() // triggers transition to half-open

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected half-open, got %s", cb.State())
	}

	// A single failure in half-open should trip back to open.
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("expected open after failure in half-open, got %s", cb.State())
	}
}

func TestCircuitBreaker_ResetForcesClosed(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: time.Minute,
	})

	// Trip the circuit open.
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected closed after Reset(), got %s", cb.State())
	}
	if !cb.Allow() {
		t.Error("expected Allow() true after Reset()")
	}
}

func TestCircuitBreaker_Defaults(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	// Should use defaults: MaxFailures=5, ResetTimeout=30s, HalfOpenMaxAttempts=3.
	// Verify MaxFailures default by recording 4 failures (should stay closed)
	// and a 5th trips it open.
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if cb.State() != StateClosed {
		t.Errorf("expected closed after 4 failures with default max=5, got %s", cb.State())
	}
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("expected open after 5 failures with default max=5, got %s", cb.State())
	}
}

func TestRTEngine_CircuitOpen(t *testing.T) {
	engine := NewEngine("test")
	rt := NewRTEngine(engine, CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: time.Minute,
	})

	rules := []Rule{
		{
			RuleID:    "rt-1",
			DatasetID: "ds1",
			RuleType:  RuleCompleteness,
			Field:     "name",
			Severity:  SeverityHigh,
			Threshold: 1.0,
			RTEnabled: true,
		},
	}

	// Force the circuit open by recording a failure directly.
	rt.cb.RecordFailure()
	if rt.CircuitState() != StateOpen {
		t.Fatalf("expected open, got %s", rt.CircuitState())
	}

	_, err := rt.EvaluateRecord(rules, Record{"name": "Alice"})
	if err == nil {
		t.Fatal("expected error when circuit is open")
	}
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestRTEngine_Success(t *testing.T) {
	engine := NewEngine("test")
	rt := NewRTEngine(engine, CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: time.Minute,
	})

	rules := []Rule{
		{
			RuleID:    "rt-completeness",
			DatasetID: "ds1",
			RuleType:  RuleCompleteness,
			Field:     "email",
			Severity:  SeverityHigh,
			Threshold: 1.0,
			RTEnabled: true,
		},
	}

	rec := Record{"email": "user@example.com"}
	result, err := rt.EvaluateRecord(rules, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Status != StatusPass {
		t.Errorf("expected pass, got %s", result.Results[0].Status)
	}
	if rt.CircuitState() != StateClosed {
		t.Errorf("expected circuit to remain closed, got %s", rt.CircuitState())
	}
}
