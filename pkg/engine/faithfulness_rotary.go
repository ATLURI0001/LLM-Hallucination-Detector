package engine

// FaithfulnessRotary tracks bounded faithfulness scores in [0, 1].
type FaithfulnessRotary struct {
	scores []float64
	cap    int
}

func NewFaithfulnessRotary(capacity int) *FaithfulnessRotary {
	return &FaithfulnessRotary{cap: capacity}
}

func (r *FaithfulnessRotary) Record(score float64) {
	if len(r.scores) >= r.cap {
		r.scores = r.scores[1:]
	}
	r.scores = append(r.scores, score)
}

func (r *FaithfulnessRotary) Avg() float64 {
	if len(r.scores) == 0 {
		return 0
	}
	var sum float64
	for _, s := range r.scores {
		sum += s
	}
	return sum / float64(len(r.scores))
}
