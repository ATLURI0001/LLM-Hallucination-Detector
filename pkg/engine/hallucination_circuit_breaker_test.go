package engine

import "testing"

func TestHallucinationCircuitBreaker(t *testing.T) {
	cb := NewHallucinationCircuitBreaker(2)
	_ = cb.Check(0.9, 0.5)
	_ = cb.Check(0.9, 0.5)
	if err := cb.Check(0.1, 0.5); err != ErrHallucinationCircuitOpen {
		t.Fatalf("expected circuit open, got %v", err)
	}
}
