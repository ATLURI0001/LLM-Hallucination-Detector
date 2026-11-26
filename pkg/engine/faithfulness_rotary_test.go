package engine

import "testing"

func TestFaithfulnessRotary(t *testing.T) {
	r := NewFaithfulnessRotary(2)
	r.Record(0.5)
	r.Record(0.8)
	r.Record(1.0)
	if got := r.Avg(); got != 0.9 {
		t.Fatalf("expected 0.9, got %v", got)
	}
}
