package engine

// GroundingLatencyRotary stores a bounded ring of grounding latency samples in ms.
type GroundingLatencyRotary struct {
	samples []int
	cap     int
}

// NewGroundingLatencyRotary creates a rotary buffer with positive capacity.
func NewGroundingLatencyRotary(capacity int) *GroundingLatencyRotary {
	if capacity <= 0 {
		capacity = 1
	}
	return &GroundingLatencyRotary{cap: capacity}
}

// Record appends a sample, evicting the oldest if at capacity.
func (r *GroundingLatencyRotary) Record(ms int) {
	if len(r.samples) >= r.cap {
		r.samples = r.samples[1:]
	}
	r.samples = append(r.samples, ms)
}

// Len returns the current number of buffered samples.
func (r *GroundingLatencyRotary) Len() int {
	return len(r.samples)
}
