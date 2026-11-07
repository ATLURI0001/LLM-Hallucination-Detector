package models

import "time"

// ClaimStatus represents the verification state of a claim against source documents.
type ClaimStatus string

const (
	StatusSupported          ClaimStatus = "SUPPORTED"
	StatusContradicted       ClaimStatus = "CONTRADICTED"
	StatusUnsupported        ClaimStatus = "UNSUPPORTED"
	StatusPartiallySupported ClaimStatus = "PARTIALLY_SUPPORTED"
)

// HallucinationType classifies the specific failure mode of an unsupported or contradicted claim.
type HallucinationType string

const (
	HallucinationNone       HallucinationType = "NONE"
	HallucinationExtrinsic  HallucinationType = "EXTRINSIC"         // Invented facts not found in source
	HallucinationIntrinsic  HallucinationType = "INTRINSIC"         // Directly contradicts source facts
	HallucinationNumeric    HallucinationType = "NUMERIC_MISMATCH"  // Numbers, dates, percentages altered
	HallucinationEntity     HallucinationType = "ENTITY_CONFUSION"  // Proper nouns or entities substituted
	HallucinationTemporal   HallucinationType = "TEMPORAL_ERROR"    // Chronological distortion
	HallucinationContextual HallucinationType = "OUT_OF_CONTEXT"    // Factual premise twisted
)

// NumericComparison represents a detected discrepancy in numbers, dates, or measurements.
type NumericComparison struct {
	ClaimToken    string `json:"claim_token"`
	EvidenceToken string `json:"evidence_token"`
	Context       string `json:"context"`
	Discrepancy   string `json:"discrepancy"`
}

// Evidence represents a grounding passage retrieved from the trusted source documents.
type Evidence struct {
	DocumentID      string  `json:"document_id"`
	Snippet         string  `json:"snippet"`
	SourceStart     int     `json:"source_start"`
	SourceEnd       int     `json:"source_end"`
	SimilarityScore float64 `json:"similarity_score"`
	ExactMatch      bool    `json:"exact_match"`
}

// Claim represents an individual atomic proposition extracted from the AI answer.
type Claim struct {
	ID                int                 `json:"id"`
	Text              string              `json:"text"`
	StartOffset       int                 `json:"start_offset"`
	EndOffset         int                 `json:"end_offset"`
	Status            ClaimStatus         `json:"status"`
	HallucinationType HallucinationType   `json:"hallucination_type"`
	Confidence        float64             `json:"confidence"`
	Explanation       string              `json:"explanation"`
	BestEvidence      *Evidence           `json:"best_evidence,omitempty"`
	Entities          []string            `json:"entities,omitempty"`
	MissingEntities   []string            `json:"missing_entities,omitempty"`
	NumericDiscrepant []NumericComparison `json:"numeric_discrepancies,omitempty"`
}

// VerificationOptions configures the detector engine.
type VerificationOptions struct {
	Threshold                float64 `json:"threshold"`                  // Overlap / confidence threshold (default: 0.65)
	StrictNumericVerification bool    `json:"strict_numeric_verification"` // Flag any number discrepancy as hallucination
	EnableLLMJudge           bool    `json:"enable_llm_judge"`           // Use LLM-as-a-judge for deep entailment
	LLMAPIKey                string  `json:"llm_api_key,omitempty"`      // API key (Gemini, OpenAI, etc.)
	LLMModel                 string  `json:"llm_model,omitempty"`        // Model identifier (e.g. gemini-2.5-flash)
	LLMBaseURL               string  `json:"llm_base_url,omitempty"`     // Custom base URL (e.g. Ollama or proxy)
	MinClaimLength           int     `json:"min_claim_length"`           // Minimum character length for valid claim
}

// DefaultVerificationOptions returns standard tuned parameters.
func DefaultVerificationOptions() VerificationOptions {
	return VerificationOptions{
		Threshold:                0.65,
		StrictNumericVerification: true,
		EnableLLMJudge:           false,
		MinClaimLength:           12,
	}
}

// VerificationResult summarizes the comprehensive hallucination detection audit.
type VerificationResult struct {
	ID                      string              `json:"id"`
	Timestamp               time.Time           `json:"timestamp"`
	AnswerText              string              `json:"answer_text"`
	SourceText              string              `json:"source_text"`
	Claims                  []Claim             `json:"claims"`
	TotalClaims             int                 `json:"total_claims"`
	SupportedClaims         int                 `json:"supported_claims"`
	ContradictedClaims      int                 `json:"contradicted_claims"`
	UnsupportedClaims       int                 `json:"unsupported_claims"`
	PartiallySupportedClaims int                 `json:"partially_supported_claims"`
	FaithfulnessScore       float64             `json:"faithfulness_score"` // 0.0 - 1.0 (higher is better)
	HallucinationScore      float64             `json:"hallucination_score"` // 0.0 - 1.0 (lower is better)
	Verdict                 string              `json:"verdict"`            // RELIABLE, NEEDS_REVIEW, UNRELIABLE
	RiskLevel               string              `json:"risk_level"`         // LOW, MEDIUM, HIGH, CRITICAL
	Summary                 string              `json:"summary"`
	ProcessingTimeMs        int64               `json:"processing_time_ms"`
	OptionsUsed             VerificationOptions `json:"options_used"`
}
