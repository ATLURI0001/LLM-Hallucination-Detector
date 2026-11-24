package claim

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidClaimID        = errors.New("claimID must be a non-empty string")
	ErrInvalidConfidenceScore = errors.New("confidence must be between 0.0 and 1.0")
	ErrInvalidVerdict        = errors.New("verdict must be VERIFIED, CONTRADICTED, or UNGROUNDED")
)

type VerificationVerdict string

const (
	VerdictVerified     VerificationVerdict = "VERIFIED"
	VerdictContradicted VerificationVerdict = "CONTRADICTED"
	VerdictUngrounded   VerificationVerdict = "UNGROUNDED"
)

// HallucinationEvent represents a single verified claim evaluation record.
type HallucinationEvent struct {
	ClaimID    string
	DocumentID string
	Verdict    VerificationVerdict
	Confidence float64
	LatencyMs  float64
	Timestamp  time.Time
}

// HallucinationTelemetrySummary captures rolling statistical metrics over a window.
type HallucinationTelemetrySummary struct {
	TotalEvaluated       int
	VerifiedCount        int
	ContradictedCount    int
	UngroundedCount      int
	HallucinationRate    float64 // (Contradicted + Ungrounded) / Total
	MeanConfidence       float64
	P50Confidence        float64
	P90Confidence        float64
	MeanLatencyMs        float64
	ActiveAlertsCount    int
	AlertMessages        []string
}

// HallucinationTelemetryRotary provides concurrent metrics collection and anomaly detection.
type HallucinationTelemetryRotary struct {
	mu                         sync.RWMutex
	maxHistory                 int
	hallucinationAlertThreshold float64 // e.g. 0.25 (25%)
	lowConfidenceThreshold     float64 // e.g. 0.60 (60%)
	events                     []HallucinationEvent
	verdictCounts              map[VerificationVerdict]int
}

// NewHallucinationTelemetryRotary initializes a new telemetry rotary.
func NewHallucinationTelemetryRotary(maxHistory int, alertRateThreshold float64, lowConfThreshold float64) *HallucinationTelemetryRotary {
	if maxHistory <= 0 {
		maxHistory = 1000
	}
	if alertRateThreshold <= 0 || alertRateThreshold > 1.0 {
		alertRateThreshold = 0.25
	}
	if lowConfThreshold <= 0 || lowConfThreshold > 1.0 {
		lowConfThreshold = 0.60
	}

	return &HallucinationTelemetryRotary{
		maxHistory:                  maxHistory,
		hallucinationAlertThreshold: alertRateThreshold,
		lowConfidenceThreshold:      lowConfThreshold,
		events:                      make([]HallucinationEvent, 0, maxHistory),
		verdictCounts:               make(map[VerificationVerdict]int),
	}
}

// RecordEvaluation records a single claim verification result into the telemetry rotary.
func (r *HallucinationTelemetryRotary) RecordEvaluation(claimID, docID string, verdict VerificationVerdict, confidence, latencyMs float64, ts time.Time) error {
	trimmedID := strings.TrimSpace(claimID)
	if trimmedID == "" {
		return ErrInvalidClaimID
	}
	if confidence < 0.0 || confidence > 1.0 || math.IsNaN(confidence) {
		return ErrInvalidConfidenceScore
	}
	if verdict != VerdictVerified && verdict != VerdictContradicted && verdict != VerdictUngrounded {
		return ErrInvalidVerdict
	}
	if ts.IsZero() {
		ts = time.Now()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	event := HallucinationEvent{
		ClaimID:    trimmedID,
		DocumentID: strings.TrimSpace(docID),
		Verdict:    verdict,
		Confidence: confidence,
		LatencyMs:  math.Max(0, latencyMs),
		Timestamp:  ts,
	}

	if len(r.events) >= r.maxHistory {
		evicted := r.events[0]
		r.events = r.events[1:]
		r.verdictCounts[evicted.Verdict]--
	}

	r.events = append(r.events, event)
	r.verdictCounts[verdict]++

	return nil
}

// GetSummary computes rolling statistical metrics across current buffered history.
func (r *HallucinationTelemetryRotary) GetSummary() HallucinationTelemetrySummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	n := len(r.events)
	if n == 0 {
		return HallucinationTelemetrySummary{}
	}

	verified := r.verdictCounts[VerdictVerified]
	contradicted := r.verdictCounts[VerdictContradicted]
	ungrounded := r.verdictCounts[VerdictUngrounded]
	hallucinated := contradicted + ungrounded

	hallucinationRate := float64(hallucinated) / float64(n)

	var sumConf, sumLat float64
	confidences := make([]float64, n)

	for i, ev := range r.events {
		sumConf += ev.Confidence
		sumLat += ev.LatencyMs
		confidences[i] = ev.Confidence
	}

	sort.Float64s(confidences)
	p50 := confidences[n/2]
	p90Idx := int(float64(n) * 0.9)
	if p90Idx >= n {
		p90Idx = n - 1
	}
	p90 := confidences[p90Idx]

	meanConf := sumConf / float64(n)
	meanLat := sumLat / float64(n)

	// Anomaly evaluation
	var alerts []string
	if n >= 10 && hallucinationRate >= r.hallucinationAlertThreshold {
		alerts = append(alerts, fmt.Sprintf("CRITICAL: Hallucination rate (%.1f%%) exceeds threshold (%.1f%%)", hallucinationRate*100, r.hallucinationAlertThreshold*100))
	}
	if n >= 10 && meanConf < r.lowConfidenceThreshold {
		alerts = append(alerts, fmt.Sprintf("WARNING: Mean verification confidence (%.2f) dropped below threshold (%.2f)", meanConf, r.lowConfidenceThreshold))
	}

	return HallucinationTelemetrySummary{
		TotalEvaluated:    n,
		VerifiedCount:     verified,
		ContradictedCount: contradicted,
		UngroundedCount:   ungrounded,
		HallucinationRate: math.Round(hallucinationRate*10000) / 10000,
		MeanConfidence:    math.Round(meanConf*10000) / 10000,
		P50Confidence:     math.Round(p50*10000) / 10000,
		P90Confidence:     math.Round(p90*10000) / 10000,
		MeanLatencyMs:     math.Round(meanLat*100) / 100,
		ActiveAlertsCount: len(alerts),
		AlertMessages:     alerts,
	}
}

// ExportPrometheusMetrics outputs metrics in standard OpenMetrics exposition format.
func (r *HallucinationTelemetryRotary) ExportPrometheusMetrics() string {
	summary := r.GetSummary()

	var sb strings.Builder
	sb.WriteString("# HELP llm_hallucination_rate_ratio Ratio of contradicted or ungrounded claims to total claims\n")
	sb.WriteString("# TYPE llm_hallucination_rate_ratio gauge\n")
	sb.WriteString(fmt.Sprintf("llm_hallucination_rate_ratio %.4f\n", summary.HallucinationRate))

	sb.WriteString("# HELP llm_claim_confidence_score_p90 P90 verification confidence score\n")
	sb.WriteString("# TYPE llm_claim_confidence_score_p90 gauge\n")
	sb.WriteString(fmt.Sprintf("llm_claim_confidence_score_p90 %.4f\n", summary.P90Confidence))

	sb.WriteString("# HELP llm_claims_verified_total Total verified claims labeled by verdict\n")
	sb.WriteString("# TYPE llm_claims_verified_total counter\n")
	sb.WriteString(fmt.Sprintf("llm_claims_verified_total{verdict=\"VERIFIED\"} %d\n", summary.VerifiedCount))
	sb.WriteString(fmt.Sprintf("llm_claims_verified_total{verdict=\"CONTRADICTED\"} %d\n", summary.ContradictedCount))
	sb.WriteString(fmt.Sprintf("llm_claims_verified_total{verdict=\"UNGROUNDED\"} %d\n", summary.UngroundedCount))

	return sb.String()
}

// Reset clears recorded events.
func (r *HallucinationTelemetryRotary) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = r.events[:0]
	r.verdictCounts = make(map[VerificationVerdict]int)
}
