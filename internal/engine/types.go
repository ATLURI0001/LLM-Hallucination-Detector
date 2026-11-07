package engine

import "github.com/user/llm-hallucination-detector/internal/models"

// Re-export core types from models package for package-level convenience
type (
	ClaimStatus         = models.ClaimStatus
	HallucinationType   = models.HallucinationType
	NumericComparison   = models.NumericComparison
	Evidence            = models.Evidence
	Claim               = models.Claim
	VerificationOptions = models.VerificationOptions
	VerificationResult  = models.VerificationResult
)

const (
	StatusSupported          = models.StatusSupported
	StatusContradicted       = models.StatusContradicted
	StatusUnsupported        = models.StatusUnsupported
	StatusPartiallySupported = models.StatusPartiallySupported

	HallucinationNone       = models.HallucinationNone
	HallucinationExtrinsic  = models.HallucinationExtrinsic
	HallucinationIntrinsic  = models.HallucinationIntrinsic
	HallucinationNumeric    = models.HallucinationNumeric
	HallucinationEntity     = models.HallucinationEntity
	HallucinationTemporal   = models.HallucinationTemporal
	HallucinationContextual = models.HallucinationContextual
)

// DefaultVerificationOptions returns standard tuned parameters.
func DefaultVerificationOptions() VerificationOptions {
	return models.DefaultVerificationOptions()
}
