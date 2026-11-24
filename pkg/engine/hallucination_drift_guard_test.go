package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestHallucinationDriftGuard_NormalToTripped(t *testing.T) {
	fakeNow := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	cfg := DefaultDriftGuardConfig()
	cfg.ConsecutiveFailureLimit = 4
	cfg.DriftThresholdPct = 50.0
	cfg.NowFunc = func() time.Time { return fakeNow }

	guard := NewHallucinationDriftGuard(cfg)

	// Record 3 grounded claims
	for i := 0; i < 3; i++ {
		state, tripped := guard.RecordClaim("claim-ok", false, 0.95)
		if state != GuardNormal || tripped {
			t.Fatalf("expected GuardNormal, got %s (tripped=%v)", state, tripped)
		}
	}

	// Record 4 consecutive hallucinations -> should trip
	for i := 0; i < 3; i++ {
		state, tripped := guard.RecordClaim("claim-bad", true, 0.99)
		if state != GuardNormal || tripped {
			t.Fatalf("expected GuardNormal before limit, got %s", state)
		}
	}
	state, tripped := guard.RecordClaim("claim-bad-4", true, 0.99)
	if state != GuardTripped || !tripped {
		t.Fatalf("expected GuardTripped, got %s (tripped=%v)", state, tripped)
	}

	snap := guard.Snapshot()
	if snap.TrippedCount != 1 || !snap.DegradedFallbackActive {
		t.Fatalf("expected tripped count 1, got %+v", snap)
	}

	// Should bypass heavy NLI while in tripped state
	if !guard.ShouldBypassHeavyNLI() {
		t.Fatalf("expected ShouldBypassHeavyNLI to return true")
	}
}

func TestHallucinationDriftGuard_CooldownAndRecovery(t *testing.T) {
	fakeNow := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	cfg := DefaultDriftGuardConfig()
	cfg.ConsecutiveFailureLimit = 2
	cfg.CooldownDuration = 5 * time.Second
	cfg.RecoveryGroundedCount = 2
	cfg.NowFunc = func() time.Time { return fakeNow }

	guard := NewHallucinationDriftGuard(cfg)

	// Trip the guard
	guard.RecordClaim("c1", true, 0.9)
	guard.RecordClaim("c2", true, 0.9)

	if guard.Snapshot().State != GuardTripped {
		t.Fatalf("expected GuardTripped")
	}

	// Advance time past cooldown
	fakeNow = fakeNow.Add(6 * time.Second)

	// Now ShouldBypassHeavyNLI should be false to allow test probes
	if guard.ShouldBypassHeavyNLI() {
		t.Fatalf("expected ShouldBypassHeavyNLI to be false after cooldown")
	}

	// Record 1 grounded claim -> transitions to Degraded
	state, _ := guard.RecordClaim("c3", false, 0.95)
	if state != GuardDegraded {
		t.Fatalf("expected GuardDegraded after 1 recovery claim, got %s", state)
	}

	// Record 2nd grounded claim -> recovers to GuardNormal
	state, _ = guard.RecordClaim("c4", false, 0.95)
	if state != GuardNormal {
		t.Fatalf("expected GuardNormal after 2 recovery claims, got %s", state)
	}
}

func TestHallucinationDriftGuard_EvaluateWithFallback(t *testing.T) {
	fakeNow := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	cfg := DefaultDriftGuardConfig()
	cfg.ConsecutiveFailureLimit = 2
	cfg.NowFunc = func() time.Time { return fakeNow }

	guard := NewHallucinationDriftGuard(cfg)
	ctx := context.Background()

	// Normal evaluation: returns evaluated result
	res, mode, err := guard.EvaluateWithFallback(ctx, "c1", func() (bool, error) {
		return false, nil
	})
	if err != nil || res != false || mode != "NEURAL_EVALUATION" {
		t.Fatalf("unexpected result: res=%v, mode=%s, err=%v", res, mode, err)
	}

	// Trip the guard
	guard.RecordClaim("bad1", true, 1.0)
	guard.RecordClaim("bad2", true, 1.0)

	// Now should use GUARD_FALLBACK_HIGH_RISK without executing heavy evaluator
	executed := false
	res, mode, err = guard.EvaluateWithFallback(ctx, "c2", func() (bool, error) {
		executed = true
		return false, nil
	})
	if executed {
		t.Fatalf("evaluator should have been bypassed")
	}
	if !res || mode != "GUARD_FALLBACK_HIGH_RISK" || err != nil {
		t.Fatalf("unexpected fallback: res=%v, mode=%s, err=%v", res, mode, err)
	}
}

func TestHallucinationDriftGuard_EvaluatorErrorFallback(t *testing.T) {
	cfg := DefaultDriftGuardConfig()
	guard := NewHallucinationDriftGuard(cfg)
	ctx := context.Background()

	// Evaluator error triggers error fallback
	expectedErr := errors.New("upstream neural timeout")
	res, mode, err := guard.EvaluateWithFallback(ctx, "err-claim", func() (bool, error) {
		return false, expectedErr
	})
	if !res || mode != "EVALUATOR_ERROR_FALLBACK" || !errors.Is(err, expectedErr) {
		t.Fatalf("expected error fallback: res=%v, mode=%s, err=%v", res, mode, err)
	}
}

func TestHallucinationDriftGuard_Concurrency(t *testing.T) {
	cfg := DefaultDriftGuardConfig()
	guard := NewHallucinationDriftGuard(cfg)

	var wg sync.WaitGroup
	workers := 8
	iterations := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				guard.RecordClaim("concurrent-claim", i%4 == 0, 0.8)
				_ = guard.Snapshot()
			}
		}(w)
	}

	wg.Wait()
	snap := guard.Snapshot()
	if snap.WindowClaims > cfg.WindowSize {
		t.Fatalf("window exceeded size: %d", snap.WindowClaims)
	}
}
