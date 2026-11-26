package engine

import (
	"errors"
	"testing"
	"time"
)

func TestVerifierPoolCircuitBreaker(t *testing.T) {
	cb := NewVerifierPoolCircuitBreaker(2, 5*time.Second)
	fail := func() error { return errors.New("pool timeout") }
	_ = cb.Call(fail)
	_ = cb.Call(fail)
	if err := cb.Call(func() error { return nil }); err != ErrVerifierCircuitOpen {
		t.Fatalf("expected ErrVerifierCircuitOpen, got %v", err)
	}
}
