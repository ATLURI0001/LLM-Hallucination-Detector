package engine

import (
	"testing"
	"time"
)

func TestClaimVerifyProfiler(t *testing.T) {
	p := NewClaimVerifyProfiler()
	p.Record(5 * time.Millisecond)
	p.Record(10 * time.Millisecond)
	if p.Count() != 2 || p.Total() != 15*time.Millisecond {
		t.Fatalf("unexpected profiler stats: count=%d total=%v", p.Count(), p.Total())
	}
}
