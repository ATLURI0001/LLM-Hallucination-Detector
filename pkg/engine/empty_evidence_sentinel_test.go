package engine

import "testing"

func TestAssertNonEmptyEvidence(t *testing.T) {
	if err := AssertNonEmptyEvidence("Water boils at 100C."); err != nil {
		t.Fatalf("expected valid evidence, got %v", err)
	}
	if err := AssertNonEmptyEvidence("   "); err == nil {
		t.Fatal("expected empty evidence sentinel violation")
	}
}
