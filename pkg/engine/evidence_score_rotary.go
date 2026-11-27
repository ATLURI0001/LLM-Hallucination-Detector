package engine

// EvidenceScoreRotary stores bounded evidence score samples.
type EvidenceScoreRotary struct {
	samples []float64
	cap     int
}

func NewEvidenceScoreRotary(cap int) *EvidenceScoreRotary {
	if cap <= 0 {
		cap = 1
	}
	return &EvidenceScoreRotary{cap: cap}
}

func (r *EvidenceScoreRotary) Record(score float64) {
	if len(r.samples) >= r.cap {
		r.samples = r.samples[1:]
	}
	r.samples = append(r.samples, score)
}

func (r *EvidenceScoreRotary) Len() int { return len(r.samples) }
