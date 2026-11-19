package claim

// ConfidenceTier categorizes numerical verification probabilities.
type ConfidenceTier string

const (
	TierHigh    ConfidenceTier = "HIGH"
	TierMedium  ConfidenceTier = "MEDIUM"
	TierLow     ConfidenceTier = "LOW"
	TierInvalid ConfidenceTier = "INVALID"
)

// ClassifyConfidence maps a 0.0-1.0 probability float to ConfidenceTier.
func ClassifyConfidence(score float64) ConfidenceTier {
	if score < 0.0 || score > 1.0 {
		return TierInvalid
	}
	if score >= 0.85 {
		return TierHigh
	}
	if score >= 0.50 {
		return TierMedium
	}
	return TierLow
}

// IsTrustworthy returns true if the score meets the high confidence threshold.
func IsTrustworthy(score float64) bool {
	return ClassifyConfidence(score) == TierHigh
}
