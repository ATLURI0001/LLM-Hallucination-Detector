package engine

import "testing"

func TestAssertClaimCountLimit(t *testing.T) {
	if err := AssertClaimCountLimit(10, 50); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := AssertClaimCountLimit(51, 50); err == nil {
		t.Fatal("expected claim overflow sentinel violation")
	}
}
