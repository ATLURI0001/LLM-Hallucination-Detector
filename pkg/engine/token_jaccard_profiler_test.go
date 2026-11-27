package engine

import "testing"

func TestBenchmarkTokenJaccard(t *testing.T) {
	s, d := BenchmarkTokenJaccard([]string{"a", "b"}, []string{"b", "c"}, 100)
	if s <= 0 || d < 0 {
		t.Fatalf("unexpected jaccard result: %v, %v", s, d)
	}
}
