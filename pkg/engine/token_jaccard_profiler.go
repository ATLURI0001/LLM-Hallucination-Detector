package engine

import "time"

// BenchmarkTokenJaccard measures token-set Jaccard overlap throughput.
func BenchmarkTokenJaccard(a, b []string, iterations int) (score float64, elapsed time.Duration) {
	start := time.Now()
	setA := make(map[string]struct{}, len(a))
	for _, w := range a {
		setA[w] = struct{}{}
	}
	for i := 0; i < iterations; i++ {
		inter := 0
		for _, w := range b {
			if _, ok := setA[w]; ok {
				inter++
			}
		}
		score = float64(inter) / float64(len(a)+len(b)-inter)
	}
	return score, time.Since(start)
}
