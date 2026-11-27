package engine

// HallucinationScoreRotary is a bounded ring buffer for recent hallucination scores.
type HallucinationScoreRotary struct {
	buf  []float64
	size int
	head int
	cnt  int
}

func NewHallucinationScoreRotary(size int) *HallucinationScoreRotary {
	return &HallucinationScoreRotary{buf: make([]float64, size), size: size}
}

func (r *HallucinationScoreRotary) Push(score float64) {
	r.buf[r.head%r.size] = score
	r.head++
	if r.cnt < r.size {
		r.cnt++
	}
}

func (r *HallucinationScoreRotary) Count() int {
	return r.cnt
}
