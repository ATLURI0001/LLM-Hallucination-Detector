package engine

import (
	"math"
	"testing"
	"time"
)

func TestVerificationMetrics(t *testing.T) {
	m := VerificationMetrics{
		TotalClaims:    10,
		VerifiedClaims: 8,
		Hallucinations: 2,
		ExecutionTime:  150 * time.Millisecond,
	}

	if math.Abs(m.HallucinationRate()-0.2) > 1e-6 {
		t.Fatalf("expected hallucination rate 0.2, got %f", m.HallucinationRate())
	}

	if math.Abs(m.VerificationSuccessRate()-0.8) > 1e-6 {
		t.Fatalf("expected verification success rate 0.8, got %f", m.VerificationSuccessRate())
	}

	// Zero claims edge case
	zeroM := VerificationMetrics{}
	if zeroM.HallucinationRate() != 0.0 {
		t.Fatal("expected 0.0 for zero claims hallucination rate")
	}
	if zeroM.VerificationSuccessRate() != 0.0 {
		t.Fatal("expected 0.0 for zero claims success rate")
	}
}
