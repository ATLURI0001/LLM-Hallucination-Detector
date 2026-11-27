package engine

import "testing"

func TestVerifierCircuitBreaker(t *testing.T) {
	b := NewVerifierCircuitBreaker(2)
	b.RecordFailure()
	b.RecordFailure()
	if !b.IsOpen() {
		t.Fatal("expected circuit breaker to open")
	}
}
