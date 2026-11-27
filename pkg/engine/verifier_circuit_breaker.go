package engine

// VerifierCircuitBreaker trips after consecutive claim verifier failures.
type VerifierCircuitBreaker struct {
	failures  int
	threshold int
	open      bool
}

func NewVerifierCircuitBreaker(threshold int) *VerifierCircuitBreaker {
	return &VerifierCircuitBreaker{threshold: threshold}
}

func (b *VerifierCircuitBreaker) RecordFailure() {
	b.failures++
	if b.failures >= b.threshold {
		b.open = true
	}
}

func (b *VerifierCircuitBreaker) IsOpen() bool {
	return b.open
}
