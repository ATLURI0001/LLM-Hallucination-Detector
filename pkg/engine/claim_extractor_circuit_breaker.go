package engine

import (
	"errors"
	"time"
)

var ErrExtractorCircuitOpen = errors.New("claim extractor circuit breaker open")

// ClaimExtractorCircuitBreaker guards claim extraction calls against failures.
type ClaimExtractorCircuitBreaker struct {
	threshold int
	cooldown  time.Duration
	failures  int
	openUntil time.Time
}

func NewClaimExtractorCircuitBreaker(threshold int, cooldown time.Duration) *ClaimExtractorCircuitBreaker {
	return &ClaimExtractorCircuitBreaker{threshold: threshold, cooldown: cooldown}
}

func (cb *ClaimExtractorCircuitBreaker) Call(fn func() error) error {
	if time.Now().Before(cb.openUntil) {
		return ErrExtractorCircuitOpen
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
