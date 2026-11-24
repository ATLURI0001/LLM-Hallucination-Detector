package engine

import (
	"context"
	"errors"
	"sync"
	"time"
)

// GuardState represents the active defense level of the hallucination drift guard.
type GuardState string

const (
	GuardNormal   GuardState = "NORMAL"
	GuardDegraded GuardState = "DEGRADED"
	GuardTripped  GuardState = "TRIPPED"
)

var (
	// ErrGuardTripped is returned when operations are rejected due to an active trip state.
	ErrGuardTripped = errors.New("hallucination drift guard is tripped; high-risk payload threshold exceeded")
)

// DriftGuardConfig specifies thresholds for drift detection and automated fallback.
type DriftGuardConfig struct {
	WindowSize              int           `json:"window_size"`
	DriftThresholdPct       float64       `json:"drift_threshold_pct"`
	ConsecutiveFailureLimit int           `json:"consecutive_failure_limit"`
	CooldownDuration        time.Duration `json:"cooldown_duration"`
	RecoveryGroundedCount   int           `json:"recovery_grounded_count"`
	NowFunc                 func() time.Time
}

// DefaultDriftGuardConfig returns production-ready resilient defaults.
func DefaultDriftGuardConfig() DriftGuardConfig {
	return DriftGuardConfig{
		WindowSize:              20,
		DriftThresholdPct:       50.0, // 50% hallucination rate trips breaker
		ConsecutiveFailureLimit: 5,    // 5 consecutive hallucinations trips breaker
		CooldownDuration:        10 * time.Second,
		RecoveryGroundedCount:   3,
		NowFunc:                 time.Now,
	}
}

type claimRecord struct {
	timestamp       time.Time
	claimID         string
	isHallucination bool
	confidence      float64
}

// DriftGuardSnapshot provides a live telemetry overview of the drift guard.
type DriftGuardSnapshot struct {
	State                       GuardState `json:"state"`
	CurrentHallucinationRatePct float64    `json:"current_hallucination_rate_pct"`
	WindowClaims                int        `json:"window_claims"`
	WindowHallucinations        int        `json:"window_hallucinations"`
	ConsecutiveHallucinations   int        `json:"consecutive_hallucinations"`
	TrippedCount                int64      `json:"tripped_count"`
	DegradedFallbackActive      bool       `json:"degraded_fallback_active"`
	LastTrippedAt               time.Time  `json:"last_tripped_at"`
	ConsecutiveGrounded         int        `json:"consecutive_grounded"`
}

// HallucinationDriftGuard protects inference pipelines from runaway hallucination bursts.
type HallucinationDriftGuard struct {
	mu                        sync.RWMutex
	config                    DriftGuardConfig
	state                     GuardState
	history                   []claimRecord
	consecutiveHallucinations int
	consecutiveGrounded       int
	trippedCount              int64
	lastTrippedAt             time.Time
}

// NewHallucinationDriftGuard constructs a new thread-safe drift guard.
func NewHallucinationDriftGuard(cfg DriftGuardConfig) *HallucinationDriftGuard {
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 20
	}
	if cfg.DriftThresholdPct <= 0 {
		cfg.DriftThresholdPct = 50.0
	}
	if cfg.ConsecutiveFailureLimit <= 0 {
		cfg.ConsecutiveFailureLimit = 5
	}
	if cfg.CooldownDuration <= 0 {
		cfg.CooldownDuration = 10 * time.Second
	}
	if cfg.RecoveryGroundedCount <= 0 {
		cfg.RecoveryGroundedCount = 3
	}
	if cfg.NowFunc == nil {
		cfg.NowFunc = time.Now
	}

	return &HallucinationDriftGuard{
		config:  cfg,
		state:   GuardNormal,
		history: make([]claimRecord, 0, cfg.WindowSize),
	}
}

// RecordClaim records an evaluated claim outcome and updates the drift state machine.
func (g *HallucinationDriftGuard) RecordClaim(claimID string, isHallucination bool, confidence float64) (GuardState, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.config.NowFunc()

	// Append to sliding window
	g.history = append(g.history, claimRecord{
		timestamp:       now,
		claimID:         claimID,
		isHallucination: isHallucination,
		confidence:      confidence,
	})

	if len(g.history) > g.config.WindowSize {
		g.history = g.history[len(g.history)-g.config.WindowSize:]
	}

	if isHallucination {
		g.consecutiveHallucinations++
		g.consecutiveGrounded = 0
	} else {
		g.consecutiveGrounded++
		g.consecutiveHallucinations = 0
	}

	// Calculate current rate in window
	hallucinationCount := 0
	for _, rec := range g.history {
		if rec.isHallucination {
			hallucinationCount++
		}
	}
	rate := (float64(hallucinationCount) / float64(len(g.history))) * 100.0

	minSamplesForRate := g.config.WindowSize / 2
	if minSamplesForRate < 4 {
		minSamplesForRate = 4
	}
	rateExceeded := len(g.history) >= minSamplesForRate && rate > g.config.DriftThresholdPct

	trippedNow := false

	// State machine transition
	switch g.state {
	case GuardTripped:
		if now.Sub(g.lastTrippedAt) >= g.config.CooldownDuration {
			if g.consecutiveGrounded >= g.config.RecoveryGroundedCount {
				g.state = GuardNormal
				g.history = g.history[:0] // Reset window on recovery
			} else if g.consecutiveGrounded > 0 {
				g.state = GuardDegraded
			}
		}
	case GuardDegraded:
		if g.consecutiveGrounded >= g.config.RecoveryGroundedCount {
			g.state = GuardNormal
			g.history = g.history[:0]
		} else if g.consecutiveHallucinations >= g.config.ConsecutiveFailureLimit || rateExceeded {
			g.state = GuardTripped
			g.lastTrippedAt = now
			g.trippedCount++
			trippedNow = true
		}
	case GuardNormal:
		if g.consecutiveHallucinations >= g.config.ConsecutiveFailureLimit || rateExceeded {
			g.state = GuardTripped
			g.lastTrippedAt = now
			g.trippedCount++
			trippedNow = true
		}
	}

	return g.state, trippedNow
}

// ShouldBypassHeavyNLI returns whether heavy neural inference should be bypassed for fallback.
func (g *HallucinationDriftGuard) ShouldBypassHeavyNLI() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	now := g.config.NowFunc()
	if g.state == GuardTripped {
		if now.Sub(g.lastTrippedAt) >= g.config.CooldownDuration {
			// Ready to attempt degraded trial
			return false
		}
		return true
	}
	return false
}

// EvaluateWithFallback safely executes an expensive NLI evaluation or uses conservative fallback.
func (g *HallucinationDriftGuard) EvaluateWithFallback(
	ctx context.Context,
	claimID string,
	heavyEvaluator func() (bool, error),
) (isHallucination bool, mode string, err error) {
	if g.ShouldBypassHeavyNLI() {
		// Defensive fallback: assume hallucination to protect downstream consumers
		g.RecordClaim(claimID, true, 1.0)
		return true, "GUARD_FALLBACK_HIGH_RISK", nil
	}

	// Execute evaluation with context check
	select {
	case <-ctx.Done():
		return true, "CONTEXT_CANCELED_FALLBACK", ctx.Err()
	default:
	}

	evalResult, evalErr := heavyEvaluator()
	if evalErr != nil {
		// On evaluator failure, record as hallucination to trigger defensive backoff
		g.RecordClaim(claimID, true, 0.5)
		return true, "EVALUATOR_ERROR_FALLBACK", evalErr
	}

	g.RecordClaim(claimID, evalResult, 0.95)
	return evalResult, "NEURAL_EVALUATION", nil
}

// Snapshot returns a copy of current guard telemetry.
func (g *HallucinationDriftGuard) Snapshot() DriftGuardSnapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()

	hallucinationCount := 0
	for _, rec := range g.history {
		if rec.isHallucination {
			hallucinationCount++
		}
	}

	var rate float64
	if len(g.history) > 0 {
		rate = (float64(hallucinationCount) / float64(len(g.history))) * 100.0
	}

	return DriftGuardSnapshot{
		State:                       g.state,
		CurrentHallucinationRatePct: rate,
		WindowClaims:                len(g.history),
		WindowHallucinations:        hallucinationCount,
		ConsecutiveHallucinations:   g.consecutiveHallucinations,
		TrippedCount:                g.trippedCount,
		DegradedFallbackActive:      g.state != GuardNormal,
		LastTrippedAt:               g.lastTrippedAt,
		ConsecutiveGrounded:         g.consecutiveGrounded,
	}
}

// Reset clears history and returns to GuardNormal.
func (g *HallucinationDriftGuard) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.state = GuardNormal
	g.history = g.history[:0]
	g.consecutiveHallucinations = 0
	g.consecutiveGrounded = 0
}
