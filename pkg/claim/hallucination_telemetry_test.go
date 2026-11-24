package claim

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHallucinationTelemetryRotary_RecordAndSummary(t *testing.T) {
	rotary := NewHallucinationTelemetryRotary(100, 0.25, 0.60)

	// Valid entries
	now := time.Now()
	err := rotary.RecordEvaluation("claim-1", "doc-1", VerdictVerified, 0.95, 12.0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = rotary.RecordEvaluation("claim-2", "doc-1", VerdictVerified, 0.90, 15.0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = rotary.RecordEvaluation("claim-3", "doc-1", VerdictContradicted, 0.85, 20.0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = rotary.RecordEvaluation("claim-4", "doc-2", VerdictUngrounded, 0.40, 10.0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	summary := rotary.GetSummary()
	if summary.TotalEvaluated != 4 {
		t.Errorf("expected 4 evaluated, got %d", summary.TotalEvaluated)
	}
	if summary.VerifiedCount != 2 {
		t.Errorf("expected 2 verified, got %d", summary.VerifiedCount)
	}
	if summary.ContradictedCount != 1 {
		t.Errorf("expected 1 contradicted, got %d", summary.ContradictedCount)
	}
	if summary.UngroundedCount != 1 {
		t.Errorf("expected 1 ungrounded, got %d", summary.UngroundedCount)
	}

	// Hallucination rate: (1 + 1) / 4 = 0.50
	if summary.HallucinationRate != 0.50 {
		t.Errorf("expected hallucination rate 0.50, got %f", summary.HallucinationRate)
	}
}

func TestHallucinationTelemetryRotary_ValidationErrors(t *testing.T) {
	rotary := NewHallucinationTelemetryRotary(100, 0.25, 0.60)
	now := time.Now()

	// Empty claim ID
	if err := rotary.RecordEvaluation("", "doc-1", VerdictVerified, 0.9, 10, now); !errors.Is(err, ErrInvalidClaimID) {
		t.Errorf("expected ErrInvalidClaimID, got %v", err)
	}

	// Out of bounds confidence
	if err := rotary.RecordEvaluation("c1", "doc-1", VerdictVerified, 1.5, 10, now); !errors.Is(err, ErrInvalidConfidenceScore) {
		t.Errorf("expected ErrInvalidConfidenceScore, got %v", err)
	}
	if err := rotary.RecordEvaluation("c1", "doc-1", VerdictVerified, -0.1, 10, now); !errors.Is(err, ErrInvalidConfidenceScore) {
		t.Errorf("expected ErrInvalidConfidenceScore, got %v", err)
	}

	// Invalid verdict
	if err := rotary.RecordEvaluation("c1", "doc-1", "UNKNOWN_VERDICT", 0.5, 10, now); !errors.Is(err, ErrInvalidVerdict) {
		t.Errorf("expected ErrInvalidVerdict, got %v", err)
	}
}

func TestHallucinationTelemetryRotary_WindowEviction(t *testing.T) {
	// Rotary with capacity 3
	rotary := NewHallucinationTelemetryRotary(3, 0.25, 0.60)
	now := time.Now()

	_ = rotary.RecordEvaluation("c1", "doc", VerdictVerified, 0.9, 10, now)
	_ = rotary.RecordEvaluation("c2", "doc", VerdictVerified, 0.9, 10, now)
	_ = rotary.RecordEvaluation("c3", "doc", VerdictContradicted, 0.8, 10, now)

	s1 := rotary.GetSummary()
	if s1.TotalEvaluated != 3 || s1.VerifiedCount != 2 || s1.ContradictedCount != 1 {
		t.Errorf("expected 3 total, 2 verified, 1 contradicted")
	}

	// 4th event evicts c1 (verified)
	_ = rotary.RecordEvaluation("c4", "doc", VerdictContradicted, 0.8, 10, now)
	s2 := rotary.GetSummary()
	if s2.TotalEvaluated != 3 {
		t.Errorf("expected total 3 after eviction, got %d", s2.TotalEvaluated)
	}
	if s2.VerifiedCount != 1 {
		t.Errorf("expected 1 verified remaining, got %d", s2.VerifiedCount)
	}
	if s2.ContradictedCount != 2 {
		t.Errorf("expected 2 contradicted, got %d", s2.ContradictedCount)
	}
}

func TestHallucinationTelemetryRotary_AnomalyAlerts(t *testing.T) {
	rotary := NewHallucinationTelemetryRotary(50, 0.20, 0.70)
	now := time.Now()

	// Record 10 events with 4 contradicted (40% hallucination rate) and 0.50 confidence
	for i := 0; i < 6; i++ {
		_ = rotary.RecordEvaluation("c-ok", "doc", VerdictVerified, 0.60, 10, now)
	}
	for i := 0; i < 4; i++ {
		_ = rotary.RecordEvaluation("c-bad", "doc", VerdictContradicted, 0.40, 10, now)
	}

	summary := rotary.GetSummary()
	if summary.ActiveAlertsCount < 2 {
		t.Errorf("expected at least 2 active alerts (high rate and low confidence), got %d: %v", summary.ActiveAlertsCount, summary.AlertMessages)
	}
}

func TestHallucinationTelemetryRotary_PrometheusExportAndReset(t *testing.T) {
	rotary := NewHallucinationTelemetryRotary(50, 0.25, 0.60)
	now := time.Now()

	_ = rotary.RecordEvaluation("c1", "doc", VerdictVerified, 0.95, 10, now)
	_ = rotary.RecordEvaluation("c2", "doc", VerdictContradicted, 0.80, 12, now)

	prom := rotary.ExportPrometheusMetrics()
	if !strings.Contains(prom, "llm_hallucination_rate_ratio 0.5000") {
		t.Errorf("expected hallucination rate metric in output: %s", prom)
	}
	if !strings.Contains(prom, "llm_claims_verified_total{verdict=\"VERIFIED\"} 1") {
		t.Errorf("expected verified counter metric in output: %s", prom)
	}

	rotary.Reset()
	if rotary.GetSummary().TotalEvaluated != 0 {
		t.Errorf("expected 0 events after reset")
	}
}

func TestHallucinationTelemetryRotary_Concurrency(t *testing.T) {
	rotary := NewHallucinationTelemetryRotary(1000, 0.25, 0.60)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			verdict := VerdictVerified
			if idx%3 == 0 {
				verdict = VerdictContradicted
			}
			_ = rotary.RecordEvaluation("c-concurr", "doc", verdict, 0.85, 10.0, time.Now())
			_ = rotary.GetSummary()
		}(i)
	}

	wg.Wait()
	summary := rotary.GetSummary()
	if summary.TotalEvaluated != 50 {
		t.Errorf("expected 50 concurrently recorded events, got %d", summary.TotalEvaluated)
	}
}
