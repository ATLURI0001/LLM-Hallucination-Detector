package engine

import "testing"

func TestEvidenceScoreRotary(t *testing.T) {
	r := NewEvidenceScoreRotary(2)
	r.Record(0.9)
	r.Record(0.7)
	r.Record(0.5)
	if r.Len() != 2 {
		t.Fatalf("expected 2 samples, got %d", r.Len())
	}
}
