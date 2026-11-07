package api

import "github.com/ATLURI0001/llm-hallucination-detector/internal/models"

// VerifyRequest payload for the /api/v1/verify endpoint
type VerifyRequest struct {
	AnswerText string                     `json:"answer_text"`
	SourceText string                     `json:"source_text"`
	Options    *models.VerificationOptions `json:"options,omitempty"`
}

// ExtractClaimsRequest payload for the /api/v1/extract endpoint
type ExtractClaimsRequest struct {
	Text           string `json:"text"`
	MinClaimLength int    `json:"min_claim_length,omitempty"`
}

// HealthResponse payload for /api/v1/health
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Uptime    string `json:"uptime"`
}

// ErrorResponse payload for API errors
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}
