package engine

import "time"

// VerificationMetrics captures latency and outcome statistics for verification runs.
type VerificationMetrics struct {
	TotalClaims      int           `json:"total_claims"`
	VerifiedClaims   int           `json:"verified_claims"`
	Hallucinations   int           `json:"hallucinations"`
	ExecutionTime    time.Duration `json:"execution_time"`
}

// HallucinationRate computes the ratio of unsupported claims to total evaluated claims.
func (m VerificationMetrics) HallucinationRate() float64 {
	if m.TotalClaims == 0 {
		return 0.0
	}
	return float64(m.Hallucinations) / float64(m.TotalClaims)
}

// VerificationSuccessRate computes the ratio of verified true claims.
func (m VerificationMetrics) VerificationSuccessRate() float64 {
	if m.TotalClaims == 0 {
		return 0.0
	}
	return float64(m.VerifiedClaims) / float64(m.TotalClaims)
}
