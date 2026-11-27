package engine

import (
	"errors"
	"testing"
	"time"
)

func TestClaimExtractorCircuitBreaker(t *testing.T) {
	cb := NewClaimExtractorCircuitBreaker(2, 5*time.Second)
	fail := func() error { return errors.New("extract err") }
	_ = cb.Call(fail)
	_ = cb.Call(fail)
	if err := cb.Call(func() error { return nil }); err != ErrExtractorCircuitOpen {
		t.Fatalf("expected ErrExtractorCircuitOpen, got %v", err)
	}
}
