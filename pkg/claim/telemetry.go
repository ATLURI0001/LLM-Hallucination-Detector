package claim

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// TelemetrySnapshot represents a point-in-time metrics report for claim verifications.
type TelemetrySnapshot struct {
	TotalVerifications uint64            `json:"total_verifications"`
	VerifiedClaims     uint64            `json:"verified_claims"`
	HallucinatedClaims uint64            `json:"hallucinated_claims"`
	RefutedClaims      uint64            `json:"refuted_claims"`
	ErrorCount         uint64            `json:"error_count"`
	HallucinationRatio float64           `json:"hallucination_ratio"`
	MeanDurationMs     float64           `json:"mean_duration_ms"`
	StdDevDurationMs   float64           `json:"std_dev_duration_ms"`
	P50DurationMs      float64           `json:"p50_duration_ms"`
	P90DurationMs      float64           `json:"p90_duration_ms"`
	P95DurationMs      float64           `json:"p95_duration_ms"`
	P99DurationMs      float64           `json:"p99_duration_ms"`
	ModelStats         map[string]uint64 `json:"model_stats"`
	Timestamp          time.Time         `json:"timestamp"`
}

// VerificationTelemetry tracks verification latency, hallucination drift, and performance anomalies.
type VerificationTelemetry struct {
	mu                 sync.RWMutex
	windowSize         int
	anomalyThreshold   float64
	totalVerifications uint64
	verifiedClaims     uint64
	hallucinatedClaims uint64
	refutedClaims      uint64
	errorCount         uint64
	modelCounts        map[string]uint64
	recentDurations    []float64
	headIndex          int
	count              int
}

// NewVerificationTelemetry initializes a thread-safe verification telemetry collector.
func NewVerificationTelemetry(windowSize int, anomalyThreshold float64) *VerificationTelemetry {
	if windowSize <= 0 {
		windowSize = 1000
	}
	if anomalyThreshold <= 0.0 {
		anomalyThreshold = 3.0
	}
	return &VerificationTelemetry{
		windowSize:       windowSize,
		anomalyThreshold: anomalyThreshold,
		modelCounts:      make(map[string]uint64),
		recentDurations:  make([]float64, windowSize),
	}
}

// RecordVerification records execution latency, model identity, and claim classification verdict.
func (t *VerificationTelemetry) RecordVerification(model string, durationMs float64, verdict string, isError bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalVerifications++
	if isError {
		t.errorCount++
	}

	normVerdict := strings.ToUpper(strings.TrimSpace(verdict))
	switch normVerdict {
	case "VERIFIED", "SUPPORTED":
		t.verifiedClaims++
	case "HALLUCINATED", "UNSUPPORTED":
		t.hallucinatedClaims++
	case "REFUTED", "CONTRADICTED":
		t.refutedClaims++
	}

	if model != "" {
		t.modelCounts[model]++
	}

	t.recentDurations[t.headIndex] = durationMs
	t.headIndex = (t.headIndex + 1) % t.windowSize
	if t.count < t.windowSize {
		t.count++
	}
}

// DetectLatencyAnomaly tests whether an observed duration constitutes a statistical outlier.
func (t *VerificationTelemetry) DetectLatencyAnomaly(durationMs float64) (isAnomaly bool, zScore float64, mean float64, stdDev float64) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.count < 10 {
		return false, 0.0, 0.0, 0.0
	}

	var sum float64
	for i := 0; i < t.count; i++ {
		sum += t.recentDurations[i]
	}
	mean = sum / float64(t.count)

	var varSum float64
	for i := 0; i < t.count; i++ {
		diff := t.recentDurations[i] - mean
		varSum += diff * diff
	}
	rawStdDev := math.Sqrt(varSum / float64(t.count))
	floor := math.Max(0.05*mean, 0.1)
	if rawStdDev < floor {
		stdDev = floor
	} else {
		stdDev = rawStdDev
	}

	zScore = (durationMs - mean) / stdDev
	isAnomaly = zScore > t.anomalyThreshold
	return isAnomaly, zScore, mean, stdDev
}

// Snapshot returns a point-in-time immutable aggregation of verification telemetry.
func (t *VerificationTelemetry) Snapshot() TelemetrySnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	models := make(map[string]uint64, len(t.modelCounts))
	for k, v := range t.modelCounts {
		models[k] = v
	}

	var ratio float64
	totalEvaluated := t.verifiedClaims + t.hallucinatedClaims + t.refutedClaims
	if totalEvaluated > 0 {
		ratio = float64(t.hallucinatedClaims) / float64(totalEvaluated)
	}

	snap := TelemetrySnapshot{
		TotalVerifications: t.totalVerifications,
		VerifiedClaims:     t.verifiedClaims,
		HallucinatedClaims: t.hallucinatedClaims,
		RefutedClaims:      t.refutedClaims,
		ErrorCount:         t.errorCount,
		HallucinationRatio: ratio,
		ModelStats:         models,
		Timestamp:          time.Now().UTC(),
	}

	if t.count == 0 {
		return snap
	}

	durations := make([]float64, t.count)
	copy(durations, t.recentDurations[:t.count])
	sort.Float64s(durations)

	var sum float64
	for _, d := range durations {
		sum += d
	}
	snap.MeanDurationMs = sum / float64(t.count)

	var varSum float64
	for _, d := range durations {
		diff := d - snap.MeanDurationMs
		varSum += diff * diff
	}
	snap.StdDevDurationMs = math.Sqrt(varSum / float64(t.count))

	snap.P50DurationMs = computePercentile(durations, 50.0)
	snap.P90DurationMs = computePercentile(durations, 90.0)
	snap.P95DurationMs = computePercentile(durations, 95.0)
	snap.P99DurationMs = computePercentile(durations, 99.0)

	return snap
}

func computePercentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0.0
	}
	if p <= 0.0 {
		return sorted[0]
	}
	if p >= 100.0 {
		return sorted[len(sorted)-1]
	}
	idx := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return sorted[lower]
	}
	weight := idx - float64(lower)
	return sorted[lower]*(1.0-weight) + sorted[upper]*weight
}

// ExportPrometheus exports verification metrics in Prometheus exposition format.
func (t *VerificationTelemetry) ExportPrometheus() string {
	snap := t.Snapshot()

	var sb strings.Builder
	sb.WriteString("# HELP hallucidetect_verifications_total Total number of verification operations executed.\n")
	sb.WriteString("# TYPE hallucidetect_verifications_total counter\n")
	sb.WriteString(fmt.Sprintf("hallucidetect_verifications_total %d\n", snap.TotalVerifications))

	sb.WriteString("# HELP hallucidetect_claims_total Claims evaluated by verdict category.\n")
	sb.WriteString("# TYPE hallucidetect_claims_total counter\n")
	sb.WriteString(fmt.Sprintf("hallucidetect_claims_total{verdict=\"verified\"} %d\n", snap.VerifiedClaims))
	sb.WriteString(fmt.Sprintf("hallucidetect_claims_total{verdict=\"hallucinated\"} %d\n", snap.HallucinatedClaims))
	sb.WriteString(fmt.Sprintf("hallucidetect_claims_total{verdict=\"refuted\"} %d\n", snap.RefutedClaims))

	sb.WriteString("# HELP hallucidetect_hallucination_ratio Ratio of hallucinated claims over total evaluated.\n")
	sb.WriteString("# TYPE hallucidetect_hallucination_ratio gauge\n")
	sb.WriteString(fmt.Sprintf("hallucidetect_hallucination_ratio %.4f\n", snap.HallucinationRatio))

	sb.WriteString("# HELP hallucidetect_duration_ms Verification duration percentiles in milliseconds.\n")
	sb.WriteString("# TYPE hallucidetect_duration_ms gauge\n")
	sb.WriteString(fmt.Sprintf("hallucidetect_duration_ms{quantile=\"0.50\"} %.2f\n", snap.P50DurationMs))
	sb.WriteString(fmt.Sprintf("hallucidetect_duration_ms{quantile=\"0.90\"} %.2f\n", snap.P90DurationMs))
	sb.WriteString(fmt.Sprintf("hallucidetect_duration_ms{quantile=\"0.95\"} %.2f\n", snap.P95DurationMs))
	sb.WriteString(fmt.Sprintf("hallucidetect_duration_ms{quantile=\"0.99\"} %.2f\n", snap.P99DurationMs))

	if len(snap.ModelStats) > 0 {
		sb.WriteString("# HELP hallucidetect_model_verifications_total Total verifications partitioned by generator model.\n")
		sb.WriteString("# TYPE hallucidetect_model_verifications_total counter\n")
		for model, count := range snap.ModelStats {
			sb.WriteString(fmt.Sprintf("hallucidetect_model_verifications_total{model=\"%s\"} %d\n", model, count))
		}
	}

	return sb.String()
}
