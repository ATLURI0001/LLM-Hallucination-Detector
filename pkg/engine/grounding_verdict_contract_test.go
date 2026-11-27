package engine

import "testing"

func TestValidateGroundingVerdict(t *testing.T) {
	if err := ValidateGroundingVerdict("supported"); err != nil {
		t.Fatalf("expected valid verdict, got %v", err)
	}
	if err := ValidateGroundingVerdict("maybe"); err == nil {
		t.Fatal("expected verdict contract violation")
	}
}
