package edqlite

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState string

const (
	StateClosed   CircuitState = "closed"
	StateOpen     CircuitState = "open"
	StateHalfOpen CircuitState = "half-open"
)

// CircuitBreakerConfig holds tuning parameters for a CircuitBreaker.
type CircuitBreakerConfig struct {
	MaxFailures         int
	ResetTimeout        time.Duration
	HalfOpenMaxAttempts int
}

// CircuitBreaker implements the circuit breaker pattern for near-real-time
// evaluation calls. It trips open after repeated failures and gradually
// recovers through a half-open probe phase.
type CircuitBreaker struct {
	mu                  sync.Mutex
	state               CircuitState
	failureCount        int
	successCount        int
	maxFailures         int
	halfOpenMaxAttempts int
	resetTimeout        time.Duration
	lastFailure         time.Time
}

// NewCircuitBreaker creates a CircuitBreaker with the given configuration.
// Zero-value fields in cfg are replaced with sensible defaults:
// MaxFailures=5, ResetTimeout=30s, HalfOpenMaxAttempts=3.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = 30 * time.Second
	}
	if cfg.HalfOpenMaxAttempts <= 0 {
		cfg.HalfOpenMaxAttempts = 3
	}
	return &CircuitBreaker{
		state:               StateClosed,
		maxFailures:         cfg.MaxFailures,
		resetTimeout:        cfg.ResetTimeout,
		halfOpenMaxAttempts: cfg.HalfOpenMaxAttempts,
	}
}

// Allow reports whether the next call should be permitted.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailure) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case StateHalfOpen:
		return cb.successCount < cb.halfOpenMaxAttempts
	default:
		return false
	}
}

// RecordSuccess records a successful call and may close the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxAttempts {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

// RecordFailure records a failed call and may trip the circuit open.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.maxFailures {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.successCount = 0
	}
}

// State returns the current circuit state in a thread-safe manner.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset forces the circuit breaker back to the closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastFailure = time.Time{}
}

// ---------------------------------------------------------------------------
// RTEngine — near-real-time engine with circuit-breaker protection.
// ---------------------------------------------------------------------------

// ErrCircuitOpen is returned when the circuit breaker is open and the call
// is not permitted.
var ErrCircuitOpen = errors.New("circuit breaker open")

// RTEngine wraps an Engine with circuit-breaker protection for near-real-time
// record-level evaluations.
type RTEngine struct {
	engine *Engine
	cb     *CircuitBreaker
}

// NewRTEngine creates an RTEngine backed by the given Engine and circuit
// breaker configuration.
func NewRTEngine(engine *Engine, cfg CircuitBreakerConfig) *RTEngine {
	return &RTEngine{
		engine: engine,
		cb:     NewCircuitBreaker(cfg),
	}
}

// EvaluateRecord runs rules against a single record, respecting the circuit
// breaker. If the circuit is open the call is rejected immediately with
// ErrCircuitOpen.
func (rt *RTEngine) EvaluateRecord(rules []Rule, rec Record) (*RunResult, error) {
	if !rt.cb.Allow() {
		return nil, ErrCircuitOpen
	}

	result, err := rt.engine.EvaluateRecord(rules, rec)
	if err != nil {
		rt.cb.RecordFailure()
		return nil, err
	}

	rt.cb.RecordSuccess()
	return result, nil
}

// CircuitState returns the current state of the underlying circuit breaker.
func (rt *RTEngine) CircuitState() CircuitState {
	return rt.cb.State()
}

// ResetCircuit forces the circuit breaker back to the closed state.
func (rt *RTEngine) ResetCircuit() {
	rt.cb.Reset()
}
