package engine

// ContradictionRateRotary tracks bounded contradiction rate samples.
type ContradictionRateRotary struct {
	samples []float64
	cap     int
}

func NewContradictionRateRotary(capacity int) *ContradictionRateRotary {
	return &ContradictionRateRotary{cap: capacity}
}

func (r *ContradictionRateRotary) Record(rate float64) {
	if len(r.samples) >= r.cap {
		r.samples = r.samples[1:]
	}
	r.samples = append(r.samples, rate)
}

func (r *ContradictionRateRotary) Avg() float64 {
	if len(r.samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range r.samples {
		sum += s
	}
	return sum / float64(len(r.samples))
}
