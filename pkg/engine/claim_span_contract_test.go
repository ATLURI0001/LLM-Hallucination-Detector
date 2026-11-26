package engine

import "testing"

func TestValidateClaimSpan(t *testing.T) {
	if err := ValidateClaimSpan(ClaimSpan{Text: "earth", Start: 0, End: 5}, 10); err != nil {
		t.Fatalf("expected valid span, got %v", err)
	}
	if err := ValidateClaimSpan(ClaimSpan{Text: "earth", Start: 5, End: 20}, 10); err == nil {
		t.Fatal("expected out-of-bounds error")
	}
}
