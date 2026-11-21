package claim

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrInvalidQueryID          = errors.New("query ID must be an alphanumeric string between 3 and 64 characters")
	ErrClaimTooShort           = errors.New("claim text must contain at least 5 characters")
	ErrNoReferenceContexts     = errors.New("at least one reference context must be provided for hallucination verification")
	ErrExcessiveContexts       = errors.New("cannot exceed maximum limit of 50 reference contexts")
	ErrInvalidThreshold        = errors.New("confidence threshold must be a float between 0.0 and 1.0")
	ErrInvalidCheckMode        = errors.New("check mode must be one of: STRICT, CONSERVATIVE, RELAXED")
)

var (
	queryIDRegex     = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)
	citationRegex    = regexp.MustCompile(`\[(?:\d+(?:\s*,\s*\d+)*|citation\s+needed)\]`)
	markdownSymbols  = regexp.MustCompile(`[*_~` + "`" + `]`)
	multiSpaceRegex  = regexp.MustCompile(`[ \t]+`)
	multiLineRegex   = regexp.MustCompile(`\n{3,}`)
)

// CheckMode governs strictness of contradiction and grounding penalties.
type CheckMode string

const (
	ModeStrict       CheckMode = "STRICT"
	ModeConservative CheckMode = "CONSERVATIVE"
	ModeRelaxed      CheckMode = "RELAXED"
)

// VerificationRequestSpec defines the ingress contract for hallucination verification.
type VerificationRequestSpec struct {
	QueryID             string    `json:"query_id"`
	ClaimText           string    `json:"claim_text"`
	ReferenceContexts   []string  `json:"reference_contexts"`
	ConfidenceThreshold float64   `json:"confidence_threshold,omitempty"`
	CheckMode           CheckMode `json:"check_mode,omitempty"`
}

// CleanedVerificationSpec represents the normalized, validated request.
type CleanedVerificationSpec struct {
	QueryID             string    `json:"query_id"`
	NormalizedClaim     string    `json:"normalized_claim"`
	RawClaim            string    `json:"raw_claim"`
	SanitizedContexts   []string  `json:"sanitized_contexts"`
	ConfidenceThreshold float64   `json:"confidence_threshold"`
	CheckMode           CheckMode `json:"check_mode"`
}

// SanitizeClaimText strips citation footnotes, markdown decorations, and control characters.
func SanitizeClaimText(raw string) string {
	// 1. Strip control characters
	var sb strings.Builder
	for _, r := range raw {
		if !unicode.IsControl(r) || r == '\n' || r == '\t' {
			sb.WriteRune(r)
		}
	}
	text := sb.String()

	// 2. Strip bracketed citation references: [1], [2, 3], [citation needed]
	text = citationRegex.ReplaceAllString(text, "")

	// 3. Strip basic markdown inline styling (*, **, _, ~~)
	text = markdownSymbols.ReplaceAllString(text, "")

	// 4. Compress horizontal whitespace
	text = multiSpaceRegex.ReplaceAllString(text, " ")
	text = multiLineRegex.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// SanitizeContextText normalizes reference text while preserving sentence and paragraph structures.
func SanitizeContextText(raw string) string {
	var sb strings.Builder
	for _, r := range raw {
		if !unicode.IsControl(r) || r == '\n' || r == '\t' {
			sb.WriteRune(r)
		}
	}
	text := multiSpaceRegex.ReplaceAllString(sb.String(), " ")
	text = multiLineRegex.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// ValidateVerificationRequest validates and canonicalizes an incoming verification spec.
func ValidateVerificationRequest(spec VerificationRequestSpec) (*CleanedVerificationSpec, error) {
	// QueryID
	qID := strings.TrimSpace(spec.QueryID)
	if !queryIDRegex.MatchString(qID) {
		return nil, ErrInvalidQueryID
	}

	// ClaimText
	cleanedClaim := SanitizeClaimText(spec.ClaimText)
	if len(cleanedClaim) < 5 {
		return nil, ErrClaimTooShort
	}
	if len(cleanedClaim) > 4096 {
		return nil, ErrClaimTooLong
	}

	// ReferenceContexts
	if len(spec.ReferenceContexts) == 0 {
		return nil, ErrNoReferenceContexts
	}
	if len(spec.ReferenceContexts) > 50 {
		return nil, ErrExcessiveContexts
	}

	sanitizedContexts := make([]string, 0, len(spec.ReferenceContexts))
	for i, ctx := range spec.ReferenceContexts {
		cClean := SanitizeContextText(ctx)
		if len(cClean) < 5 {
			return nil, fmt.Errorf("reference context at index %d is too short (< 5 chars)", i)
		}
		sanitizedContexts = append(sanitizedContexts, cClean)
	}

	// ConfidenceThreshold
	thresh := spec.ConfidenceThreshold
	if thresh == 0.0 {
		thresh = 0.70
	} else if thresh < 0.0 || thresh > 1.0 {
		return nil, ErrInvalidThreshold
	}

	// CheckMode
	mode := spec.CheckMode
	if mode == "" {
		mode = ModeConservative
	} else if mode != ModeStrict && mode != ModeConservative && mode != ModeRelaxed {
		return nil, ErrInvalidCheckMode
	}

	return &CleanedVerificationSpec{
		QueryID:             qID,
		NormalizedClaim:     cleanedClaim,
		RawClaim:            spec.ClaimText,
		SanitizedContexts:   sanitizedContexts,
		ConfidenceThreshold: thresh,
		CheckMode:           mode,
	}, nil
}
