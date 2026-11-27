package engine

import "time"

// ClaimVerifyProfiler tracks claim verification durations.
type ClaimVerifyProfiler struct {
	samples []time.Duration
}

func NewClaimVerifyProfiler() *ClaimVerifyProfiler {
	return &ClaimVerifyProfiler{}
}

func (p *ClaimVerifyProfiler) Record(d time.Duration) {
	p.samples = append(p.samples, d)
}

func (p *ClaimVerifyProfiler) Count() int {
	return len(p.samples)
}

func (p *ClaimVerifyProfiler) Total() time.Duration {
	var sum time.Duration
	for _, s := range p.samples {
		sum += s
	}
	return sum
}
