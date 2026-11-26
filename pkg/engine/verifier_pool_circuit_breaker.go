package engine

import (
	"errors"
	"time"
)

var ErrVerifierCircuitOpen = errors.New("verifier pool circuit breaker open")

// VerifierPoolCircuitBreaker protects verification worker pool calls.
type VerifierPoolCircuitBreaker struct {
	threshold int
	cooldown  time.Duration
	failures  int
	openUntil time.Time
}

func NewVerifierPoolCircuitBreaker(threshold int, cooldown time.Duration) *VerifierPoolCircuitBreaker {
	return &VerifierPoolCircuitBreaker{threshold: threshold, cooldown: cooldown}
}

func (cb *VerifierPoolCircuitBreaker) Call(fn func() error) error {
	if time.Now().Before(cb.openUntil) {
		return ErrVerifierCircuitOpen
	}
	if err := fn(); err != nil {
		cb.failures++
		if cb.failures >= cb.threshold {
			cb.openUntil = time.Now().Add(cb.cooldown)
			cb.failures = 0
		}
		return err
	}
	cb.failures = 0
	return nil
}
