package engine

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// VerificationSample represents a single claim verification telemetry event.
type VerificationSample struct {
	ClaimID         string
	Latency         time.Duration
	Confidence      float64
	Entropy         float64
	EntailmentState string
	Timestamp       time.Time
}

// LatencyPercentiles holds percentile calculation results.
type LatencyPercentiles struct {
	P50 time.Duration
	P90 time.Duration
	P99 time.Duration
	Min time.Duration
	Max time.Duration
	Avg time.Duration
}

// UncertaintySummary provides aggregated diagnostics for claims.
type UncertaintySummary struct {
	TotalSamples      int
	ContradictionRate float64
	AvgConfidence     float64
	AvgEntropy        float64
	Percentiles       LatencyPercentiles
	BreachCount       int
}

// UncertaintyTelemetryRotary implements a thread-safe ring buffer for claim verification telemetry.
type UncertaintyTelemetryRotary struct {
	mu                sync.RWMutex
	capacity          int
	breachThresholdMs float64
	buffer            []*VerificationSample
	headIndex         int
	totalIngested     int
}

// NewUncertaintyTelemetryRotary constructs a new telemetry rotary buffer.
func NewUncertaintyTelemetryRotary(capacity int, breachThresholdMs float64) (*UncertaintyTelemetryRotary, error) {
	if capacity < 10 {
		return nil, errors.New("capacity must be at least 10")
	}
	if breachThresholdMs <= 0 {
		return nil, errors.New("breachThresholdMs must be positive")
	}

	return &UncertaintyTelemetryRotary{
		capacity:          capacity,
		breachThresholdMs: breachThresholdMs,
		buffer:            make([]*VerificationSample, capacity),
		headIndex:         0,
		totalIngested:     0,
	}, nil
}

// RecordVerification appends a verification sample into the circular buffer in O(1).
func (r *UncertaintyTelemetryRotary) RecordVerification(sample VerificationSample) error {
	if sample.ClaimID == "" {
		return errors.New("sample ClaimID cannot be empty")
	}
	if sample.Confidence < 0.0 || sample.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got: %f", sample.Confidence)
	}
	if sample.Entropy < 0.0 {
		return fmt.Errorf("entropy cannot be negative, got: %f", sample.Entropy)
	}
	if sample.Timestamp.IsZero() {
		sample.Timestamp = time.Now()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	sCopy := sample
	r.buffer[r.headIndex] = &sCopy
	r.headIndex = (r.headIndex + 1) % r.capacity
	r.totalIngested++

	return nil
}

// GetActiveSamples returns all currently populated samples in the buffer.
func (r *UncertaintyTelemetryRotary) GetActiveSamples() []VerificationSample {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := r.totalIngested
	if count > r.capacity {
		count = r.capacity
	}

	res := make([]VerificationSample, 0, count)
	for i := 0; i < count; i++ {
		if r.buffer[i] != nil {
			res = append(res, *r.buffer[i])
		}
	}
	return res
}

// Summarize aggregates metrics and percentiles over all active samples.
func (r *UncertaintyTelemetryRotary) Summarize() UncertaintySummary {
	samples := r.GetActiveSamples()
	n := len(samples)
	if n == 0 {
		return UncertaintySummary{}
	}

	durations := make([]time.Duration, n)
	var totalConfidence, totalEntropy float64
	var contradictionCount, breachCount int
	var totalDur time.Duration

	for i, s := range samples {
		durations[i] = s.Latency
		totalDur += s.Latency
		totalConfidence += s.Confidence
		totalEntropy += s.Entropy

		if s.EntailmentState == "CONTRADICTION" {
			contradictionCount++
		}
		if float64(s.Latency.Milliseconds()) >= r.breachThresholdMs {
			breachCount++
		}
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	getP := func(p float64) time.Duration {
		idx := int(math.Floor((p / 100.0) * float64(n)))
		if idx >= n {
			idx = n - 1
		}
		return durations[idx]
	}

	percentiles := LatencyPercentiles{
		P50: getP(50),
		P90: getP(90),
		P99: getP(99),
		Min: durations[0],
		Max: durations[n-1],
		Avg: totalDur / time.Duration(n),
	}

	return UncertaintySummary{
		TotalSamples:      n,
		ContradictionRate: math.Round((float64(contradictionCount)/float64(n))*1000) / 1000,
		AvgConfidence:     math.Round((totalConfidence/float64(n))*1000) / 1000,
		AvgEntropy:        math.Round((totalEntropy/float64(n))*1000) / 1000,
		Percentiles:       percentiles,
		BreachCount:       breachCount,
	}
}

// Reset clears the buffer and indices.
func (r *UncertaintyTelemetryRotary) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.buffer {
		r.buffer[i] = nil
	}
	r.headIndex = 0
	r.totalIngested = 0
}
