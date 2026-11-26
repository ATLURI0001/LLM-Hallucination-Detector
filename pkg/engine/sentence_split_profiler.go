package engine

import (
	"strings"
	"time"
)

// BenchmarkSentenceSplit measures sentence splitting throughput.
func BenchmarkSentenceSplit(text string, iterations int) (count int, elapsed time.Duration) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		count = len(strings.Split(text, "."))
	}
	return count, time.Since(start)
}
