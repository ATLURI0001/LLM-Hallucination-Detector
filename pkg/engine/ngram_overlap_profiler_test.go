package engine

import "testing"

func TestNgramOverlapProfiler(t *testing.T) {
	ratio := BigramOverlapRatio("the quick brown fox", "the quick brown dog")
	if ratio <= 0.5 {
		t.Fatalf("expected >0.5 overlap, got %f", ratio)
	}

	p := &NgramOverlapProfiler{}
	tps := p.RunBenchmark("the quick brown fox", "the quick brown dog", 200)
	if tps <= 0 {
		t.Fatalf("expected positive throughput, got %d", tps)
	}
}
