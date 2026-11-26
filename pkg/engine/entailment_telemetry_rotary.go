package engine

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"
)

var (
	ErrInvalidCapacity = errors.New("rotary: capacity must be greater than zero")
)

// EntailmentObservation captures inference metrics for a verified claim.
type EntailmentObservation struct {
	ClaimID            string
	EntailmentScore    float64
	ContradictionScore float64
	NeutralScore       float64
	LatencyMs          int64
	Timestamp          time.Time
}

// EntailmentTelemetrySummary encapsulates statistical aggregates across observations.
type EntailmentTelemetrySummary struct {
	SampleCount            int
	MeanEntailment         float64
	StdDevEntailment       float64
	HallucinationRatio     float64
	P50LatencyMs           int64
	P90LatencyMs           int64
	P99LatencyMs           int64
	MeanConfidenceEntropy float64
}

// EntailmentTelemetryRotary is a thread-safe circular buffer for monitoring claim verification.
type EntailmentTelemetryRotary struct {
	mu       sync.RWMutex
	capacity int
	buffer   []EntailmentObservation
	head     int
	count    int
	total    int64
}

// NewEntailmentTelemetryRotary initializes a new telemetry rotary buffer.
func NewEntailmentTelemetryRotary(capacity int) (*EntailmentTelemetryRotary, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &EntailmentTelemetryRotary{
		capacity: capacity,
		buffer:   make([]EntailmentObservation, capacity),
	}, nil
}

// Record inserts a new observation into the ring buffer.
func (r *EntailmentTelemetryRotary) Record(obs EntailmentObservation) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if obs.Timestamp.IsZero() {
		obs.Timestamp = time.Now()
	}

	r.buffer[r.head] = obs
	r.head = (r.head + 1) % r.capacity
	if r.count < r.capacity {
		r.count++
	}
	r.total++
}

// GetSummary aggregates metrics from all current observations.
func (r *EntailmentTelemetryRotary) GetSummary() EntailmentTelemetrySummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return EntailmentTelemetrySummary{}
	}

	var sumEntailment float64
	var hallucinationCount int
	var sumEntropy float64
	latencies := make([]int64, r.count)

	for i := 0; i < r.count; i++ {
		obs := r.buffer[i]
		sumEntailment += obs.EntailmentScore
		if obs.ContradictionScore > obs.EntailmentScore {
			hallucinationCount++
		}

		// Calculate Shannon entropy over 3 classes: [entailment, contradiction, neutral]
		sumEntropy += calculateEntropy(obs.EntailmentScore, obs.ContradictionScore, obs.NeutralScore)
		latencies[i] = obs.LatencyMs
	}

	mean := sumEntailment / float64(r.count)

	// Calculate standard deviation
	var varianceSum float64
	for i := 0; i < r.count; i++ {
		diff := r.buffer[i].EntailmentScore - mean
		varianceSum += diff * diff
	}
	stdDev := math.Sqrt(varianceSum / float64(r.count))

	// Sort latencies for percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := latencies[int(float64(r.count)*0.5)]
	p90 := latencies[int(float64(r.count)*0.9)]
	p99 := latencies[int(float64(r.count)*0.99)]

	return EntailmentTelemetrySummary{
		SampleCount:            r.count,
		MeanEntailment:         math.Round(mean*10000) / 10000,
		StdDevEntailment:       math.Round(stdDev*10000) / 10000,
		HallucinationRatio:     math.Round((float64(hallucinationCount)/float64(r.count))*10000) / 10000,
		P50LatencyMs:           p50,
		P90LatencyMs:           p90,
		P99LatencyMs:           p99,
		MeanConfidenceEntropy: math.Round((sumEntropy/float64(r.count))*10000) / 10000,
	}
}

func calculateEntropy(p1, p2, p3 float64) float64 {
	probs := []float64{p1, p2, p3}
	var entropy float64
	for _, p := range probs {
		if p > 1e-9 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}
