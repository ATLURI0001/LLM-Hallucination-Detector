package engine

import "testing"

func TestGroundingLatencyRotary(t *testing.T) {
	r := NewGroundingLatencyRotary(2)
	r.Record(12)
	r.Record(18)
	r.Record(25)
	if r.Len() != 2 {
		t.Fatalf("expected len 2, got %d", r.Len())
	}
}
