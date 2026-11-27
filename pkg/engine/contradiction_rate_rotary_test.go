package engine

import "testing"

func TestContradictionRateRotary(t *testing.T) {
	r := NewContradictionRateRotary(2)
	r.Record(0.1)
	r.Record(0.2)
	r.Record(0.4)
	if got := r.Avg(); got != 0.30000000000000004 {
		t.Fatalf("expected 0.3, got %v", got)
	}
}
