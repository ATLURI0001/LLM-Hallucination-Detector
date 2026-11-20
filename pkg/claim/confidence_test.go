package claim

import "testing"

func TestClassifyConfidence(t *testing.T) {
	if ClassifyConfidence(0.95) != TierHigh {
		t.Fatal("expected 0.95 to be TierHigh")
	}
	if ClassifyConfidence(0.60) != TierMedium {
		t.Fatal("expected 0.60 to be TierMedium")
	}
	if ClassifyConfidence(0.30) != TierLow {
		t.Fatal("expected 0.30 to be TierLow")
	}
	if ClassifyConfidence(-0.1) != TierInvalid {
		t.Fatal("expected negative score to be TierInvalid")
	}
	if ClassifyConfidence(1.2) != TierInvalid {
		t.Fatal("expected score > 1.0 to be TierInvalid")
	}
}

func TestIsTrustworthy(t *testing.T) {
	if !IsTrustworthy(0.90) {
		t.Fatal("expected 0.90 to be trustworthy")
	}
	if IsTrustworthy(0.80) {
		t.Fatal("expected 0.80 not to be trustworthy")
	}
}
