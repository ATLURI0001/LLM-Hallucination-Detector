package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestEntailmentCircuitBreaker_NormalExecution(t *testing.T) {
	cb := NewEntailmentCircuitBreaker(DefaultCircuitBreakerConfig())

	score, fallback, err := cb.Execute(context.Background(), func() (float64, error) {
		return 0.94, nil
	}, 0.50)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fallback {
		t.Fatalf("expected non-fallback execution")
	}
	if score != 0.94 {
		t.Fatalf("expected score 0.94, got %f", score)
	}
	if cb.GetState() != StateClosed {
		t.Fatalf("expected state CLOSED, got %s", cb.GetState())
	}
}

func TestEntailmentCircuitBreaker_TripAndRecovery(t *testing.T) {
	currTime := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	cfg := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		TimeoutDuration:  10 * time.Second,
		NowFunc:          func() time.Time { return currTime },
	}

	cb := NewEntailmentCircuitBreaker(cfg)

	failErr := errors.New("inference timeout")
	// Trigger 3 failures to trip the circuit
	for i := 0; i < 3; i++ {
		_, fb, err := cb.Execute(context.Background(), func() (float64, error) {
			return 0.0, failErr
		}, 0.35)
		if !fb || err == nil {
			t.Fatalf("expected failure and fallback on call %d", i)
		}
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("expected state OPEN, got %s", cb.GetState())
	}

	// In OPEN state, calls should immediately return fallback score without invoking function
	called := false
	score, fb, err := cb.Execute(context.Background(), func() (float64, error) {
		called = true
		return 0.99, nil
	}, 0.40)

	if called {
		t.Fatalf("function should not have been called when circuit is OPEN")
	}
	if !fb || score != 0.40 || err != nil {
		t.Fatalf("expected fallback score 0.40 with no error, got score=%f, fb=%v, err=%v", score, fb, err)
	}

	// Advance time past timeout duration
	currTime = currTime.Add(11 * time.Second)

	// Next call should enter HALF_OPEN and invoke function
	score, fb, err = cb.Execute(context.Background(), func() (float64, error) {
		return 0.90, nil
	}, 0.50)
	if fb || err != nil || score != 0.90 {
		t.Fatalf("half-open call 1 failed: %v", err)
	}
	if cb.GetState() != StateHalfOpen {
		t.Fatalf("expected HALF_OPEN, got %s", cb.GetState())
	}

	// 2nd successful call should close circuit
	score, fb, err = cb.Execute(context.Background(), func() (float64, error) {
		return 0.92, nil
	}, 0.50)
	if fb || err != nil || score != 0.92 {
		t.Fatalf("half-open call 2 failed: %v", err)
	}
	if cb.GetState() != StateClosed {
		t.Fatalf("expected CLOSED after 2 successes, got %s", cb.GetState())
	}
}

func TestEntailmentCircuitBreaker_ContextCancellation(t *testing.T) {
	cb := NewEntailmentCircuitBreaker(DefaultCircuitBreakerConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled context

	_, fb, err := cb.Execute(ctx, func() (float64, error) {
		time.Sleep(50 * time.Millisecond)
		return 0.85, nil
	}, 0.20)

	if !fb {
		t.Fatalf("expected fallback on cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}
}

func TestEntailmentCircuitBreaker_DetectAndResolveCycles(t *testing.T) {
	cb := NewEntailmentCircuitBreaker(DefaultCircuitBreakerConfig())

	// Cyclic graph: A -> B -> C -> A
	graph := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {"A", "D"},
		"D": {},
	}

	order, pruned, err := cb.DetectAndResolveCycles(graph, []string{"A"})
	if err != nil {
		t.Fatalf("unexpected error during cycle resolution: %v", err)
	}
	if !pruned {
		t.Fatalf("expected cycle to be detected and pruned")
	}
	if len(order) != 4 {
		t.Fatalf("expected 4 topologically sorted nodes, got %d (%v)", len(order), order)
	}

	// C's neighbor "A" should have been pruned to prevent deadlock loop
	for _, nbr := range graph["C"] {
		if nbr == "A" {
			t.Fatalf("cycle edge C -> A was not pruned")
		}
	}
}

func TestEntailmentCircuitBreaker_MaxDepthGuard(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxGraphDepth: 3,
	}
	cb := NewEntailmentCircuitBreaker(cfg)

	// Chain exceeding depth 3: N1 -> N2 -> N3 -> N4 -> N5
	deepGraph := map[string][]string{
		"N1": {"N2"},
		"N2": {"N3"},
		"N3": {"N4"},
		"N4": {"N5"},
		"N5": {},
	}

	_, _, err := cb.DetectAndResolveCycles(deepGraph, []string{"N1"})
	if !errors.Is(err, ErrMaxGraphDepthExceeded) {
		t.Fatalf("expected ErrMaxGraphDepthExceeded, got: %v", err)
	}
}

func TestEntailmentCircuitBreaker_ManualTripAndReset(t *testing.T) {
	cb := NewEntailmentCircuitBreaker(DefaultCircuitBreakerConfig())
	cb.Trip()
	if cb.GetState() != StateOpen {
		t.Fatalf("expected StateOpen after Trip()")
	}
	cb.Reset()
	if cb.GetState() != StateClosed {
		t.Fatalf("expected StateClosed after Reset()")
	}
}

func TestEntailmentCircuitBreaker_Concurrency(t *testing.T) {
	cb := NewEntailmentCircuitBreaker(DefaultCircuitBreakerConfig())
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				_, _, _ = cb.Execute(context.Background(), func() (float64, error) {
					if (idx+j)%5 == 0 {
						return 0.0, errors.New("flaky engine")
					}
					return 0.88, nil
				}, 0.50)
			}
		}(i)
	}

	wg.Wait()
	stats := cb.Stats()
	if stats.TotalRequests != 500 {
		t.Fatalf("expected 500 requests, got %d", stats.TotalRequests)
	}
}
