package claim

import (
	"errors"
	"math"
	"sort"
)

const (
	defaultAlpha       = 0.05
	defaultTemperature = 1.0
	logitEpsilon       = 1e-7
)

// ConformalCalibrator establishes finite-sample statistical guarantees for claim verification scores
// using inductive conformal prediction bounds and post-hoc temperature scaling.
type ConformalCalibrator struct {
	scores      []float64 // Sorted non-conformity calibration scores in ascending order
	alpha       float64   // Significance level (error rate), e.g. 0.05 for 95% coverage
	temperature float64   // Temperature scaling factor for logits (T > 0)
}

// NewConformalCalibrator constructs a ConformalCalibrator from calibration non-conformity scores.
func NewConformalCalibrator(calibrationScores []float64, alpha float64, temperature float64) (*ConformalCalibrator, error) {
	if len(calibrationScores) == 0 {
		return nil, errors.New("calibration scores cannot be empty")
	}

	if alpha <= 0.0 || alpha >= 1.0 {
		alpha = defaultAlpha
	}
	if temperature <= 0.0 || math.IsNaN(temperature) || math.IsInf(temperature, 0) {
		temperature = defaultTemperature
	}

	// Filter and copy valid finite scores in [0.0, 1.0]
	cleaned := make([]float64, 0, len(calibrationScores))
	for _, s := range calibrationScores {
		if !math.IsNaN(s) && !math.IsInf(s, 0) {
			if s < 0.0 {
				s = 0.0
			} else if s > 1.0 {
				s = 1.0
			}
			cleaned = append(cleaned, s)
		}
	}

	if len(cleaned) == 0 {
		return nil, errors.New("no valid finite calibration scores provided")
	}

	sort.Float64s(cleaned)

	return &ConformalCalibrator{
		scores:      cleaned,
		alpha:       alpha,
		temperature: temperature,
	}, nil
}

// QuantileThreshold computes the empirical (1 - alpha) conformal prediction quantile:
// q_hat = score at index ceil((n + 1) * (1 - alpha)) - 1 (1-based index mapped to 0-based).
func (c *ConformalCalibrator) QuantileThreshold() float64 {
	n := float64(len(c.scores))
	// Standard conformal prediction index calculation: ceil((n + 1) * (1 - alpha))
	idx := int(math.Ceil((n + 1.0) * (1.0 - c.alpha))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(c.scores) {
		idx = len(c.scores) - 1
	}
	return c.scores[idx]
}

// ComputePValue computes the finite-sample valid conformal p-value for a candidate non-conformity score:
// p = (1 + count(s_i <= candidateScore)) / (n + 1)
func (c *ConformalCalibrator) ComputePValue(candidateScore float64) float64 {
	if math.IsNaN(candidateScore) {
		return 0.0
	}
	if candidateScore < 0.0 {
		candidateScore = 0.0
	} else if candidateScore > 1.0 {
		candidateScore = 1.0
	}

	n := len(c.scores)
	// Binary search count of elements <= candidateScore
	idx := sort.Search(n, func(i int) bool {
		return c.scores[i] > candidateScore
	})

	return float64(1+idx) / float64(n+1)
}

// IsCovered returns true if candidateScore falls within the (1 - alpha) confidence prediction interval.
func (c *ConformalCalibrator) IsCovered(candidateScore float64) bool {
	return candidateScore <= c.QuantileThreshold()
}

// CalibrateProbability applies temperature scaling to a raw uncalibrated probability:
// p_cal = 1 / (1 + exp(-logit(p) / T))
func (c *ConformalCalibrator) CalibrateProbability(rawProb float64) float64 {
	if math.IsNaN(rawProb) {
		return 0.5
	}
	// Clamp probability strictly away from 0 and 1 for numerical stability
	p := math.Max(logitEpsilon, math.Min(1.0-logitEpsilon, rawProb))

	// Compute logit = ln(p / (1 - p))
	logit := math.Log(p / (1.0 - p))

	// Scale by temperature
	scaledLogit := logit / c.temperature

	// Sigmoid transform
	return 1.0 / (1.0 + math.Exp(-scaledLogit))
}

// ScoreCount returns the number of active calibration samples.
func (c *ConformalCalibrator) ScoreCount() int {
	return len(c.scores)
}
