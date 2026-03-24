package edqlite

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation, requests flow through
	CircuitOpen                         // Failures exceeded threshold, requests are blocked
	CircuitHalfOpen                     // Testing if the service has recovered
)

// String returns the human-readable state name.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return fmt.Sprintf("state(%d)", int(s))
	}
}

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// MaxFailures is the number of consecutive failures before opening the circuit.
	MaxFailures int
	// ResetTimeout is how long the circuit stays open before transitioning to half-open.
	ResetTimeout time.Duration
	// HalfOpenMaxAttempts is the number of test requests allowed in half-open state.
	HalfOpenMaxAttempts int
	// OnStateChange is called when the circuit transitions between states.
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:         5,
		ResetTimeout:        30 * time.Second,
		HalfOpenMaxAttempts: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern for event delivery.
type CircuitBreaker struct {
	cfg             CircuitBreakerConfig
	state           CircuitState
	failures        int
	successes       int
	halfOpenAttempts int
	lastFailure     time.Time
	mu              sync.Mutex
	totalTripped    atomic.Int64
	totalBlocked    atomic.Int64
	totalSuccesses  atomic.Int64
	totalFailures   atomic.Int64
}

// NewCircuitBreaker creates a circuit breaker with the given config.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = 30 * time.Second
	}
	if cfg.HalfOpenMaxAttempts <= 0 {
		cfg.HalfOpenMaxAttempts = 1
	}
	return &CircuitBreaker{
		cfg:   cfg,
		state: CircuitClosed,
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// Execute runs the given function through the circuit breaker.
// Returns ErrCircuitOpen if the circuit is open and the timeout hasn't elapsed.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		cb.totalBlocked.Add(1)
		return ErrCircuitOpen
	}

	err := fn()
	cb.recordResult(err)
	return err
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if we should transition from open to half-open
	if cb.state == CircuitOpen && time.Since(cb.lastFailure) >= cb.cfg.ResetTimeout {
		cb.transition(CircuitHalfOpen)
	}
	return cb.state
}

// Reset forces the circuit back to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transition(CircuitClosed)
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenAttempts = 0
}

// Stats returns circuit breaker statistics.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.Lock()
	state := cb.state
	failures := cb.failures
	cb.mu.Unlock()

	return CircuitBreakerStats{
		State:              state.String(),
		ConsecutiveFailures: failures,
		TotalTripped:       cb.totalTripped.Load(),
		TotalBlocked:       cb.totalBlocked.Load(),
		TotalSuccesses:     cb.totalSuccesses.Load(),
		TotalFailures:      cb.totalFailures.Load(),
	}
}

// CircuitBreakerStats contains circuit breaker statistics.
type CircuitBreakerStats struct {
	State               string `json:"state"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	TotalTripped        int64  `json:"total_tripped"`
	TotalBlocked        int64  `json:"total_blocked"`
	TotalSuccesses      int64  `json:"total_successes"`
	TotalFailures       int64  `json:"total_failures"`
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) >= cb.cfg.ResetTimeout {
			cb.transition(CircuitHalfOpen)
			cb.halfOpenAttempts = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		if cb.halfOpenAttempts < cb.cfg.HalfOpenMaxAttempts {
			cb.halfOpenAttempts++
			return true
		}
		return false
	}
	return false
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.totalFailures.Add(1)
		cb.failures++
		cb.successes = 0
		cb.lastFailure = time.Now()

		switch cb.state {
		case CircuitClosed:
			if cb.failures >= cb.cfg.MaxFailures {
				cb.transition(CircuitOpen)
				cb.totalTripped.Add(1)
			}
		case CircuitHalfOpen:
			// Failed during probe — back to open
			cb.transition(CircuitOpen)
			cb.totalTripped.Add(1)
		}
	} else {
		cb.totalSuccesses.Add(1)
		cb.successes++

		switch cb.state {
		case CircuitClosed:
			cb.failures = 0
		case CircuitHalfOpen:
			// Successful probe — close the circuit
			cb.transition(CircuitClosed)
			cb.failures = 0
			cb.halfOpenAttempts = 0
		}
	}
}

func (cb *CircuitBreaker) transition(to CircuitState) {
	if cb.state == to {
		return
	}
	from := cb.state
	cb.state = to
	if cb.cfg.OnStateChange != nil {
		go cb.cfg.OnStateChange(from, to)
	}
}

// ProtectedHandler wraps a HandlerFunc with circuit breaker protection.
func ProtectedHandler(cb *CircuitBreaker, handler HandlerFunc) HandlerFunc {
	return func(evt Event) error {
		return cb.Execute(func() error {
			return handler(evt)
		})
	}
}

// ProtectedSubscription is a convenience for subscribing with circuit breaker protection.
func ProtectedSubscription(broker *Broker, pattern TopicPattern, cb *CircuitBreaker, handler HandlerFunc, opts ...SubscribeOption) (string, error) {
	return broker.Subscribe(pattern, ProtectedHandler(cb, handler), opts...)
}
