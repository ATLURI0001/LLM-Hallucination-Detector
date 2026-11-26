package engine

import (
	"testing"
	"time"
)

func TestUncertaintyTelemetryRotary_Nominal(t *testing.T) {
	rotary, err := NewUncertaintyTelemetryRotary(50, 40.0)
	if err != nil {
		t.Fatalf("failed to create rotary: %v", err)
	}

	sample := VerificationSample{
		ClaimID:         "claim-101",
		Latency:         15 * time.Millisecond,
		Confidence:      0.92,
		Entropy:         0.18,
		EntailmentState: "ENTAILMENT",
		Timestamp:       time.Now(),
	}

	if err := rotary.RecordVerification(sample); err != nil {
		t.Fatalf("failed to record sample: %v", err)
	}

	active := rotary.GetActiveSamples()
	if len(active) != 1 {
		t.Fatalf("expected 1 active sample, got %d", len(active))
	}
	if active[0].ClaimID != "claim-101" {
		t.Fatalf("unexpected claim id: %s", active[0].ClaimID)
	}
}

func TestUncertaintyTelemetryRotary_Rollover(t *testing.T) {
	capacity := 10
	rotary, err := NewUncertaintyTelemetryRotary(capacity, 50.0)
	if err != nil {
		t.Fatalf("failed to create rotary: %v", err)
	}

	for i := 0; i < 25; i++ {
		err := rotary.RecordVerification(VerificationSample{
			ClaimID:         "claim-loop",
			Latency:         time.Duration(i) * time.Millisecond,
			Confidence:      0.8,
			Entropy:         0.2,
			EntailmentState: "ENTAILMENT",
		})
		if err != nil {
			t.Fatalf("unexpected error at %d: %v", i, err)
		}
	}

	active := rotary.GetActiveSamples()
	if len(active) != capacity {
		t.Fatalf("expected capacity %d, got %d", capacity, len(active))
	}
}

func TestUncertaintyTelemetryRotary_SummarizeAndPercentiles(t *testing.T) {
	rotary, _ := NewUncertaintyTelemetryRotary(50, 25.0)

	for i := 1; i <= 10; i++ {
		state := "ENTAILMENT"
		if i <= 3 {
			state = "CONTRADICTION"
		}
		_ = rotary.RecordVerification(VerificationSample{
			ClaimID:         "claim-test",
			Latency:         time.Duration(i*5) * time.Millisecond, // 5ms to 50ms
			Confidence:      float64(i) * 0.1,
			Entropy:         float64(i) * 0.05,
			EntailmentState: state,
		})
	}

	summary := rotary.Summarize()
	if summary.TotalSamples != 10 {
		t.Fatalf("expected 10 samples, got %d", summary.TotalSamples)
	}
	if summary.ContradictionRate != 0.3 {
		t.Fatalf("expected contradiction rate 0.3, got %f", summary.ContradictionRate)
	}
	if summary.Percentiles.Min != 5*time.Millisecond {
		t.Fatalf("expected min 5ms, got %v", summary.Percentiles.Min)
	}
	if summary.Percentiles.Max != 50*time.Millisecond {
		t.Fatalf("expected max 50ms, got %v", summary.Percentiles.Max)
	}
	if summary.BreachCount != 6 { // 25ms, 30ms, 35ms, 40ms, 45ms, 50ms >= 25ms
		t.Fatalf("expected 6 breaches, got %d", summary.BreachCount)
	}
}

func TestUncertaintyTelemetryRotary_DefensiveValidation(t *testing.T) {
	rotary, _ := NewUncertaintyTelemetryRotary(20, 50.0)

	if err := rotary.RecordVerification(VerificationSample{ClaimID: ""}); err == nil {
		t.Fatal("expected error on empty claim id")
	}

	if err := rotary.RecordVerification(VerificationSample{ClaimID: "c", Confidence: 1.5}); err == nil {
		t.Fatal("expected error on confidence > 1.0")
	}

	if err := rotary.RecordVerification(VerificationSample{ClaimID: "c", Confidence: 0.5, Entropy: -1.0}); err == nil {
		t.Fatal("expected error on negative entropy")
	}

	if _, err := NewUncertaintyTelemetryRotary(5, 50.0); err == nil {
		t.Fatal("expected error on capacity < 10")
	}
}
