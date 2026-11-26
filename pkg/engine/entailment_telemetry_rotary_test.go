package engine

import (
	"errors"
	"testing"
	"time"
)

func TestEntailmentTelemetryRotary_InitValidation(t *testing.T) {
	_, err := NewEntailmentTelemetryRotary(0)
	if !errors.Is(err, ErrInvalidCapacity) {
		t.Fatalf("expected ErrInvalidCapacity, got %v", err)
	}

	rotary, err := NewEntailmentTelemetryRotary(10)
	if err != nil {
		t.Fatalf("unexpected error creating rotary: %v", err)
	}
	summary := rotary.GetSummary()
	if summary.SampleCount != 0 {
		t.Fatalf("expected empty summary sample count 0, got %d", summary.SampleCount)
	}
}

func TestEntailmentTelemetryRotary_WrapAroundAndMetrics(t *testing.T) {
	rotary, err := NewEntailmentTelemetryRotary(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Insert 10 observations to force wrap-around
	for i := 1; i <= 10; i++ {
		isHallucination := i%2 == 0
		entailment := 0.8
		contradiction := 0.1
		if isHallucination {
			entailment = 0.1
			contradiction = 0.85
		}

		rotary.Record(EntailmentObservation{
			ClaimID:            "claim-1",
			EntailmentScore:    entailment,
			ContradictionScore: contradiction,
			NeutralScore:       0.05,
			LatencyMs:          int64(i * 10),
			Timestamp:          time.Now(),
		})
	}

	summary := rotary.GetSummary()
	if summary.SampleCount != 5 {
		t.Errorf("expected sample count 5, got %d", summary.SampleCount)
	}
	if summary.HallucinationRatio <= 0 {
		t.Errorf("expected positive hallucination ratio, got %f", summary.HallucinationRatio)
	}
	if summary.P50LatencyMs <= 0 {
		t.Errorf("expected positive P50 latency, got %d", summary.P50LatencyMs)
	}
	if summary.MeanConfidenceEntropy <= 0 {
		t.Errorf("expected positive confidence entropy, got %f", summary.MeanConfidenceEntropy)
	}
}
