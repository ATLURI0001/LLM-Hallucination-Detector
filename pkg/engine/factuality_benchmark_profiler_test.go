package engine

import (
	"testing"
)

func TestVerifyClaimPair(t *testing.T) {
	profiler := NewFactualityBenchmarkProfiler()

	claim := "The company quarterly revenue grew by 15 percent"
	refValid := "According to financial reports, company quarterly revenue grew by 15 percent in Q3"
	refInvalid := "The weather in Seattle remained foggy and rainy all week"

	probSupported, isSup := profiler.VerifyClaimPair(claim, refValid)
	if !isSup || probSupported < 0.5 {
		t.Fatalf("expected claim to be supported by reference, got prob=%f isSup=%v", probSupported, isSup)
	}

	probUnsupported, isNotSup := profiler.VerifyClaimPair(claim, refInvalid)
	if isNotSup || probUnsupported > 0.2 {
		t.Fatalf("expected claim to be unsupported, got prob=%f isNotSup=%v", probUnsupported, isNotSup)
	}
}

func TestRunBenchmarkExecution(t *testing.T) {
	profiler := NewFactualityBenchmarkProfiler()

	claims := []string{
		"Tesla delivered 400000 vehicles in the fourth quarter",
		"Inflation dropped to 2.4 percent in October",
		"OpenAI launched new reasoning models with extended thinking",
	}

	references := []string{
		"Automaker Tesla reported quarterly deliveries exceeding 400000 vehicles",
		"Economic indicators showed inflation cooling to 2.4 percent in October",
		"A major hurricane made landfall along the Gulf Coast",
	}

	report := profiler.RunBenchmark(claims, references, 4, 50)

	if report.TotalEvaluations != 3*3*50 {
		t.Fatalf("expected 450 evaluations, got %d", report.TotalEvaluations)
	}
	if report.Concurrency != 4 {
		t.Fatalf("expected concurrency 4, got %d", report.Concurrency)
	}
	if report.ThroughputOpsPerSec <= 0 {
		t.Fatalf("expected positive throughput, got %f", report.ThroughputOpsPerSec)
	}
	if report.TotalDuration <= 0 {
		t.Fatalf("expected positive duration")
	}
	if report.P50LatencyNs < 0 || report.P90LatencyNs < 0 {
		t.Fatalf("expected non-negative latencies")
	}
}
