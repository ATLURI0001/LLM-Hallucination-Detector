package claim

import (
	"strings"
	"sync"
	"testing"
)

func TestVerificationTelemetry_RecordAndSnapshot(t *testing.T) {
	telem := NewVerificationTelemetry(100, 2.5)

	telem.RecordVerification("gpt-4o", 25.0, "VERIFIED", false)
	telem.RecordVerification("gpt-4o", 30.0, "HALLUCINATED", false)
	telem.RecordVerification("claude-3-5", 40.0, "REFUTED", false)
	telem.RecordVerification("claude-3-5", 50.0, "VERIFIED", true) // with error

	snap := telem.Snapshot()
	if snap.TotalVerifications != 4 {
		t.Fatalf("expected 4 verifications, got %d", snap.TotalVerifications)
	}
	if snap.VerifiedClaims != 2 {
		t.Fatalf("expected 2 verified claims, got %d", snap.VerifiedClaims)
	}
	if snap.HallucinatedClaims != 1 {
		t.Fatalf("expected 1 hallucinated claim, got %d", snap.HallucinatedClaims)
	}
	if snap.RefutedClaims != 1 {
		t.Fatalf("expected 1 refuted claim, got %d", snap.RefutedClaims)
	}
	if snap.ErrorCount != 1 {
		t.Fatalf("expected 1 error count, got %d", snap.ErrorCount)
	}

	expectedRatio := 1.0 / 4.0 // 1 hallucinated out of 4 evaluated
	if snap.HallucinationRatio < expectedRatio-0.01 || snap.HallucinationRatio > expectedRatio+0.01 {
		t.Fatalf("expected hallucination ratio ~0.25, got %f", snap.HallucinationRatio)
	}
	if snap.ModelStats["gpt-4o"] != 2 {
		t.Fatalf("expected 2 gpt-4o verifications, got %d", snap.ModelStats["gpt-4o"])
	}
}

func TestVerificationTelemetry_Percentiles(t *testing.T) {
	telem := NewVerificationTelemetry(100, 3.0)

	for i := 1; i <= 100; i++ {
		telem.RecordVerification("test-model", float64(i), "VERIFIED", false)
	}

	snap := telem.Snapshot()
	if snap.P50DurationMs < 49.0 || snap.P50DurationMs > 52.0 {
		t.Errorf("expected P50 ~50.5, got %f", snap.P50DurationMs)
	}
	if snap.P90DurationMs < 89.0 || snap.P90DurationMs > 91.5 {
		t.Errorf("expected P90 ~90.1, got %f", snap.P90DurationMs)
	}
	if snap.P99DurationMs < 98.0 || snap.P99DurationMs > 100.0 {
		t.Errorf("expected P99 ~99.0, got %f", snap.P99DurationMs)
	}
}

func TestVerificationTelemetry_AnomalyDetection(t *testing.T) {
	telem := NewVerificationTelemetry(100, 2.5)

	// Fewer than 10 samples
	isAnomaly, _, _, _ := telem.DetectLatencyAnomaly(500.0)
	if isAnomaly {
		t.Fatal("expected false when count < 10")
	}

	// Baseline samples around 50ms (48-52ms)
	for i := 0; i < 20; i++ {
		telem.RecordVerification("baseline", 50.0+float64(i%5)-2.0, "VERIFIED", false)
	}

	// Normal observation
	isAnomaly, zScore, _, _ := telem.DetectLatencyAnomaly(51.0)
	if isAnomaly {
		t.Fatalf("expected 51ms not to be an anomaly, zScore=%f", zScore)
	}

	// 5x latency spike (250ms)
	isAnomaly, zScore, _, _ = telem.DetectLatencyAnomaly(250.0)
	if !isAnomaly {
		t.Fatalf("expected 250ms to be flagged as anomaly, zScore=%f", zScore)
	}
}

func TestVerificationTelemetry_PrometheusExport(t *testing.T) {
	telem := NewVerificationTelemetry(50, 3.0)
	telem.RecordVerification("gpt-4o", 20.0, "VERIFIED", false)
	telem.RecordVerification("gpt-4o", 25.0, "HALLUCINATED", false)

	prom := telem.ExportPrometheus()
	expected := []string{
		"hallucidetect_verifications_total 2",
		"hallucidetect_claims_total{verdict=\"verified\"} 1",
		"hallucidetect_claims_total{verdict=\"hallucinated\"} 1",
		"hallucidetect_hallucination_ratio 0.5000",
		"hallucidetect_duration_ms{quantile=\"0.50\"}",
		"hallucidetect_model_verifications_total{model=\"gpt-4o\"} 2",
	}

	for _, exp := range expected {
		if !strings.Contains(prom, exp) {
			t.Errorf("expected prometheus output to contain '%s', got:\n%s", exp, prom)
		}
	}
}

func TestVerificationTelemetry_Concurrency(t *testing.T) {
	telem := NewVerificationTelemetry(500, 3.0)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				telem.RecordVerification("worker", float64(j*2), "VERIFIED", false)
				_ = telem.Snapshot()
				_, _, _, _ = telem.DetectLatencyAnomaly(50.0)
			}
		}(i)
	}

	wg.Wait()
	snap := telem.Snapshot()
	if snap.TotalVerifications != 1000 {
		t.Fatalf("expected 1000 total verifications, got %d", snap.TotalVerifications)
	}
}
