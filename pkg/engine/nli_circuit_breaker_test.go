package engine

import (
	"errors"
	"testing"
	"time"
)

func TestNLICircuitBreaker_TripAndFallback(t *testing.T) {
	cb := NewNLICircuitBreaker(2, 2, 50*time.Millisecond)
	failFn := func() (float64, error) { return 0, errors.New("timeout") }
	okFn := func() (float64, error) { return 0.92, nil }

	// Failure 1 -> CLOSED
	_, fallback, _ := cb.Execute(failFn)
	if !fallback || cb.State() != "CLOSED" {
		t.Fatalf("expected fallback=true and state=CLOSED, got %v, %s", fallback, cb.State())
	}

	// Failure 2 -> OPEN
	cb.Execute(failFn)
	if cb.State() != "OPEN" {
		t.Fatalf("expected state=OPEN, got %s", cb.State())
	}

	// Short-circuit while OPEN
	score, fallback, err := cb.Execute(okFn)
	if !fallback || !errors.Is(err, ErrNLICircuitOpen) || score != 0.5 {
		t.Fatalf("expected short-circuit fallback 0.5, got %v, %v, %v", score, fallback, err)
	}

	// Wait cooldown -> HALF_OPEN -> CLOSED after 2 successes
	time.Sleep(60 * time.Millisecond)
	cb.Execute(okFn)
	cb.Execute(okFn)
	if cb.State() != "CLOSED" {
		t.Fatalf("expected state=CLOSED after recovery, got %s", cb.State())
	}
}
