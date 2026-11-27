package engine

import "testing"

func TestHallucinationScoreRotary(t *testing.T) {
	r := NewHallucinationScoreRotary(2)
	r.Push(0.1)
	r.Push(0.2)
	r.Push(0.3)
	if r.Count() != 2 {
		t.Fatalf("expected 2, got %d", r.Count())
	}
}
