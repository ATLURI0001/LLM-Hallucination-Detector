package engine

import "testing"

func TestValidateCitationIndex(t *testing.T) {
	if err := ValidateCitationIndex(1, 3); err != nil {
		t.Fatalf("expected valid index, got %v", err)
	}
	if err := ValidateCitationIndex(3, 3); err == nil {
		t.Fatal("expected out-of-range citation index error")
	}
}
