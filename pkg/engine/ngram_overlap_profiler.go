package engine

import (
	"strings"
	"time"
)

// NgramOverlapProfiler benchmarks fast bigram overlap scoring for claim grounding.
type NgramOverlapProfiler struct{}

// BigramOverlapRatio computes Jaccard bigram similarity between two strings.
func BigramOverlapRatio(claim, evidence string) float64 {
	cWords := strings.Fields(strings.ToLower(claim))
	eWords := strings.Fields(strings.ToLower(evidence))
	if len(cWords) < 2 || len(eWords) < 2 {
		return 0
	}

	eSet := make(map[string]bool, len(eWords))
	for i := 0; i < len(eWords)-1; i++ {
		eSet[eWords[i]+" "+eWords[i+1]] = true
	}

	hits := 0
	total := len(cWords) - 1
	for i := 0; i < total; i++ {
		if eSet[cWords[i]+" "+cWords[i+1]] {
			hits++
		}
	}
	return float64(hits) / float64(total)
}

// RunBenchmark profiles bigram overlap throughput over iterations.
func (p *NgramOverlapProfiler) RunBenchmark(claim, evidence string, iterations int) int64 {
	if iterations <= 0 {
		iterations = 100
	}
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = BigramOverlapRatio(claim, evidence)
	}
	elapsedNs := time.Since(start).Nanoseconds()
	if elapsedNs <= 0 {
		elapsedNs = 1
	}
	return int64(float64(iterations) / (float64(elapsedNs) / 1e9))
}
