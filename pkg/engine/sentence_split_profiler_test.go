package engine

import "testing"

func TestBenchmarkSentenceSplit(t *testing.T) {
	c, d := BenchmarkSentenceSplit("First sentence. Second sentence. Third.", 100)
	if c < 3 || d < 0 {
		t.Fatalf("unexpected result: %d, %v", c, d)
	}
}
