package engine

import (
	"context"
	"testing"
)

func TestDetectorVerification(t *testing.T) {
	opts := DefaultVerificationOptions()
	detector := NewDetector(opts)

	sourceText := `QuantumCorp announced Q3 2024 revenue of $420 million, representing a 15% increase year-over-year.
The research division successfully deployed 1,200 qubit processors in Munich, Germany.
CEO Alice Walker confirmed that operations in France were halted due to regulatory hurdles.`

	// AI Answer with a mix of supported claim, numeric hallucination, and entity hallucination
	aiAnswer := `QuantumCorp announced Q3 2024 revenue of $420 million.
The research division deployed 5,000 qubit processors in Munich.
Operations in France were successfully expanded by Elon Musk.
The company has 80,000 employees globally.`

	res, err := detector.Verify(context.Background(), aiAnswer, sourceText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalClaims != 4 {
		t.Fatalf("expected 4 claims, got %d", res.TotalClaims)
	}

	// Claim 1: Supported
	if res.Claims[0].Status != StatusSupported {
		t.Errorf("expected claim 1 to be SUPPORTED, got %s (%s)", res.Claims[0].Status, res.Claims[0].Explanation)
	}

	// Claim 2: Numeric mismatch (5,000 vs 1,200)
	if res.Claims[1].Status != StatusContradicted && res.Claims[1].Status != StatusPartiallySupported {
		t.Errorf("expected claim 2 to be CONTRADICTED or PARTIALLY_SUPPORTED, got %s", res.Claims[1].Status)
	}
	if res.Claims[1].HallucinationType != HallucinationNumeric {
		t.Errorf("expected NUMERIC_MISMATCH hallucination, got %s", res.Claims[1].HallucinationType)
	}

	// Claim 3: Entity hallucination (Elon Musk not in source) or contradiction (halted vs expanded)
	if res.Claims[2].Status == StatusSupported {
		t.Errorf("expected claim 3 to be flagged, got SUPPORTED")
	}

	// Claim 4: Extrinsic hallucination (80,000 employees nowhere in source)
	if res.Claims[3].Status != StatusUnsupported {
		t.Errorf("expected claim 4 to be UNSUPPORTED, got %s", res.Claims[3].Status)
	}

	// Verify Overall Result
	if res.Verdict == "RELIABLE" {
		t.Errorf("expected UNRELIABLE or NEEDS_REVIEW, got %s", res.Verdict)
	}
}

func TestDetectorCanceledContext(t *testing.T) {
	detector := NewDetector(DefaultVerificationOptions())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := detector.Verify(ctx, "The sky is blue and grass is green.", "The sky is blue.")
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got err=%v res=%v", err, res)
	}
}
