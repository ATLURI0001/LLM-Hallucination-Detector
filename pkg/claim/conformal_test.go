package claim

import (
	"math"
	"testing"
)

func TestConformalCalibrator_QuantileAndCoverage(t *testing.T) {
	// 10 calibration scores: 0.1, 0.2, ..., 1.0
	scores := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}
	alpha := 0.2 // 80% coverage
	cal, err := NewConformalCalibrator(scores, alpha, 1.0)
	if err != nil {
		t.Fatalf("unexpected error initializing calibrator: %v", err)
	}

	q := cal.QuantileThreshold()
	if q < 0.8 || q > 1.0 {
		t.Fatalf("expected quantile threshold ~0.9 for alpha=0.2, got %f", q)
	}

	if !cal.IsCovered(0.5) {
		t.Fatal("expected 0.5 to be covered")
	}
	if cal.IsCovered(1.5) {
		t.Fatal("expected 1.5 exceeding quantile to not be covered")
	}
}

func TestConformalCalibrator_ComputePValue(t *testing.T) {
	scores := []float64{0.2, 0.4, 0.6, 0.8}
	cal, err := NewConformalCalibrator(scores, 0.1, 1.0)
	if err != nil {
		t.Fatalf("failed init: %v", err)
	}

	// Below all scores: (1 + 0) / (4 + 1) = 0.2
	pLow := cal.ComputePValue(0.1)
	if math.Abs(pLow-0.2) > 1e-6 {
		t.Fatalf("expected p-value 0.2, got %f", pLow)
	}

	// Above all scores: (1 + 4) / 5 = 1.0
	pHigh := cal.ComputePValue(0.9)
	if math.Abs(pHigh-1.0) > 1e-6 {
		t.Fatalf("expected p-value 1.0, got %f", pHigh)
	}
}

func TestConformalCalibrator_TemperatureScaling(t *testing.T) {
	scores := []float64{0.5}
	// Temperature > 1 softens overconfident probabilities toward 0.5
	calSoft, _ := NewConformalCalibrator(scores, 0.05, 2.0)
	softProb := calSoft.CalibrateProbability(0.95)
	if softProb >= 0.95 || softProb <= 0.5 {
		t.Fatalf("expected softened probability between 0.5 and 0.95, got %f", softProb)
	}

	// Temperature < 1 sharpens probabilities away from 0.5
	calSharp, _ := NewConformalCalibrator(scores, 0.05, 0.5)
	sharpProb := calSharp.CalibrateProbability(0.95)
	if sharpProb <= 0.95 || sharpProb > 1.0 {
		t.Fatalf("expected sharpened probability > 0.95, got %f", sharpProb)
	}

	// Midpoint 0.5 remains 0.5 regardless of temperature
	midProb := calSoft.CalibrateProbability(0.5)
	if math.Abs(midProb-0.5) > 1e-5 {
		t.Fatalf("expected midpoint probability 0.5, got %f", midProb)
	}
}

func TestConformalCalibrator_ValidationErrors(t *testing.T) {
	_, err := NewConformalCalibrator([]float64{}, 0.05, 1.0)
	if err == nil {
		t.Fatal("expected error for empty calibration scores")
	}

	// All NaN scores
	_, err = NewConformalCalibrator([]float64{math.NaN()}, 0.05, 1.0)
	if err == nil {
		t.Fatal("expected error when all scores are NaN")
	}
}
