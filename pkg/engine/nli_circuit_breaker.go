package engine

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNLICircuitOpen = errors.New("nli: inference circuit breaker is open, falling back to heuristic verifier")
)

// NLICircuitBreaker protects the NLI model inference endpoint from cascading timeouts.
type NLICircuitBreaker struct {
	mu                   sync.RWMutex
	failureThreshold     int
	recoveryThreshold    int
	cooldownDuration     time.Duration
	state                string // CLOSED, OPEN, HALF_OPEN
	consecutiveFailures  int
	consecutiveSuccesses int
	tripTime             time.Time
}

// NewNLICircuitBreaker initializes a circuit breaker for NLI inference calls.
func NewNLICircuitBreaker(failureThreshold int, recoveryThreshold int, cooldown time.Duration) *NLICircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if recoveryThreshold <= 0 {
		recoveryThreshold = 2
	}
	if cooldown <= 0 {
		cooldown = 1 * time.Second
	}
	return &NLICircuitBreaker{
		failureThreshold:  failureThreshold,
		recoveryThreshold: recoveryThreshold,
		cooldownDuration:  cooldown,
		state:             "CLOSED",
	}
}

// Execute wraps an NLI inference function with circuit breaking and heuristic fallback.
func (cb *NLICircuitBreaker) Execute(fn func() (float64, error)) (float64, bool, error) {
	cb.mu.Lock()
	if cb.state == "OPEN" {
		if time.Since(cb.tripTime) >= cb.cooldownDuration {
			cb.state = "HALF_OPEN"
			cb.consecutiveSuccesses = 0
		} else {
			cb.mu.Unlock()
			return 0.5, true, ErrNLICircuitOpen // Fallback neutral confidence
		}
	}
	cb.mu.Unlock()

	score, err := fn()
	if err != nil {
		cb.recordFailure()
		return 0.5, true, err
	}

	cb.recordSuccess()
	return score, false, nil
}

func (cb *NLICircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFailures++
	cb.consecutiveSuccesses = 0
	if cb.state == "HALF_OPEN" || cb.consecutiveFailures >= cb.failureThreshold {
		cb.state = "OPEN"
		cb.tripTime = time.Now()
	}
}

func (cb *NLICircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == "HALF_OPEN" {
		cb.consecutiveSuccesses++
		if cb.consecutiveSuccesses >= cb.recoveryThreshold {
			cb.state = "CLOSED"
			cb.consecutiveFailures = 0
		}
	} else {
		cb.consecutiveFailures = 0
	}
}

// State returns current circuit breaker state.
func (cb *NLICircuitBreaker) State() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
