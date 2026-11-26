package engine

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
)

var (
	ErrInvalidProbabilitySum = errors.New("probability distribution sum must equal 1.0 +/- epsilon")
	ErrNegativeProbability   = errors.New("probabilities must be non-negative and bounded in [0.0, 1.0]")
	ErrEmptyPremise          = errors.New("premise text cannot be empty or whitespace-only")
	ErrEmptyClaim            = errors.New("claim text cannot be empty or whitespace-only")
	ErrPremiseTooLong        = errors.New("premise text exceeds maximum allowed length of 50000 characters")
	ErrClaimTooLong          = errors.New("claim text exceeds maximum allowed length of 2000 characters")
)

var (
	invisibleCharsRegex = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F\x{200B}-\x{200D}\x{FEFF}]`)
	promptDelimRegex    = regexp.MustCompile(`(?i)<\|(?:im_start|im_end|endoftext|system|user|assistant)\|>`)
)

// EntailmentVerdict represents the canonical boundary determination for an NLI prediction.
type EntailmentVerdict string

const (
	VerdictEntailed  EntailmentVerdict = "ENTAILED"
	VerdictContradicted EntailmentVerdict = "CONTRADICTED"
	VerdictNeutral   EntailmentVerdict = "NEUTRAL"
	VerdictAmbiguous EntailmentVerdict = "AMBIGUOUS"
	VerdictRejected  EntailmentVerdict = "REJECTED"
)

// EntailmentPredictionInput holds candidate NLI probabilities for premise-claim evaluation.
type EntailmentPredictionInput struct {
	Premise        string  `json:"premise"`
	Claim          string  `json:"claim"`
	PEntailment    float64 `json:"p_entailment"`
	PNeutral       float64 `json:"p_neutral"`
	PContradiction float64 `json:"p_contradiction"`
}

// EntailmentContractConfig configures contract invariants and confidence floors.
type EntailmentContractConfig struct {
	ProbabilityEpsilon     float64 `json:"probability_epsilon"`
	ConfidenceFloor        float64 `json:"confidence_floor"`
	MinConfidenceMargin    float64 `json:"min_confidence_margin"`
	MaxPremiseLengthChars  int     `json:"max_premise_length_chars"`
	MaxClaimLengthChars    int     `json:"max_claim_length_chars"`
}

// DefaultEntailmentContractConfig returns standard production invariants.
func DefaultEntailmentContractConfig() EntailmentContractConfig {
	return EntailmentContractConfig{
		ProbabilityEpsilon:    0.015,
		ConfidenceFloor:       0.50,
		MinConfidenceMargin:   0.15,
		MaxPremiseLengthChars: 50000,
		MaxClaimLengthChars:   2000,
	}
}

// EntailmentContractEvaluation contains the validated verdict, confidence, and audit trail.
type EntailmentContractEvaluation struct {
	Verdict           EntailmentVerdict `json:"verdict"`
	SanitizedPremise  string            `json:"sanitized_premise"`
	SanitizedClaim    string            `json:"sanitized_claim"`
	TopLabel          string            `json:"top_label"`
	TopConfidence     float64           `json:"top_confidence"`
	ConfidenceMargin  float64           `json:"confidence_margin"`
	IsValid           bool              `json:"is_valid"`
	AuditReason       string            `json:"audit_reason"`
}

// EntailmentBoundaryContract enforces schema invariants on NLI predictions.
type EntailmentBoundaryContract struct {
	config EntailmentContractConfig
}

// NewEntailmentBoundaryContract creates a contract validator with configuration.
func NewEntailmentBoundaryContract(cfg *EntailmentContractConfig) *EntailmentBoundaryContract {
	if cfg == nil {
		defaultCfg := DefaultEntailmentContractConfig()
		cfg = &defaultCfg
	}
	return &EntailmentBoundaryContract{config: *cfg}
}

// SanitizeText strips invisible unicode and prompt delimiters from inputs.
func SanitizeText(input string) string {
	cleaned := invisibleCharsRegex.ReplaceAllString(input, "")
	cleaned = promptDelimRegex.ReplaceAllString(cleaned, "")
	return strings.TrimSpace(cleaned)
}

// ValidateAndEvaluate enforces distribution invariants and computes canonical verdict.
func (c *EntailmentBoundaryContract) ValidateAndEvaluate(input EntailmentPredictionInput) (EntailmentContractEvaluation, error) {
	// 1. Text sanitization & length checks
	cleanPremise := SanitizeText(input.Premise)
	if cleanPremise == "" {
		return EntailmentContractEvaluation{Verdict: VerdictRejected, AuditReason: "empty premise"}, ErrEmptyPremise
	}
	if len(cleanPremise) > c.config.MaxPremiseLengthChars {
		return EntailmentContractEvaluation{Verdict: VerdictRejected, AuditReason: "premise exceeds length cap"}, ErrPremiseTooLong
	}

	cleanClaim := SanitizeText(input.Claim)
	if cleanClaim == "" {
		return EntailmentContractEvaluation{Verdict: VerdictRejected, AuditReason: "empty claim"}, ErrEmptyClaim
	}
	if len(cleanClaim) > c.config.MaxClaimLengthChars {
		return EntailmentContractEvaluation{Verdict: VerdictRejected, AuditReason: "claim exceeds length cap"}, ErrClaimTooLong
	}

	// 2. Probability bounds check
	pe := input.PEntailment
	pn := input.PNeutral
	pc := input.PContradiction

	if pe < 0.0 || pe > 1.0 || pn < 0.0 || pn > 1.0 || pc < 0.0 || pc > 1.0 ||
		math.IsNaN(pe) || math.IsNaN(pn) || math.IsNaN(pc) ||
		math.IsInf(pe, 0) || math.IsInf(pn, 0) || math.IsInf(pc, 0) {
		return EntailmentContractEvaluation{Verdict: VerdictRejected, AuditReason: "probabilities outside [0..1] or non-finite"}, ErrNegativeProbability
	}

	// 3. Sum of probabilities invariant check
	sum := pe + pn + pc
	if math.Abs(sum-1.0) > c.config.ProbabilityEpsilon {
		return EntailmentContractEvaluation{
			Verdict:     VerdictRejected,
			AuditReason: fmt.Sprintf("probability sum %.4f violates unity invariant (eps: %.4f)", sum, c.config.ProbabilityEpsilon),
		}, ErrInvalidProbabilitySum
	}

	// 4. Identify top and runner-up probabilities
	type labelProb struct {
		label string
		prob  float64
	}
	probs := []labelProb{
		{"entailment", pe},
		{"neutral", pn},
		{"contradiction", pc},
	}

	// Simple sort of 3 elements descending
	if probs[0].prob < probs[1].prob {
		probs[0], probs[1] = probs[1], probs[0]
	}
	if probs[1].prob < probs[2].prob {
		probs[1], probs[2] = probs[2], probs[1]
	}
	if probs[0].prob < probs[1].prob {
		probs[0], probs[1] = probs[1], probs[0]
	}

	top := probs[0]
	runnerUp := probs[1]
	margin := top.prob - runnerUp.prob

	var verdict EntailmentVerdict
	var reason string

	// 5. Margin and Confidence Floor Evaluation
	if top.prob < c.config.ConfidenceFloor {
		verdict = VerdictAmbiguous
		reason = fmt.Sprintf("top label '%s' confidence %.2f below floor %.2f", top.label, top.prob, c.config.ConfidenceFloor)
	} else if margin < c.config.MinConfidenceMargin {
		verdict = VerdictAmbiguous
		reason = fmt.Sprintf("confidence margin %.2f between '%s' and '%s' below threshold %.2f", margin, top.label, runnerUp.label, c.config.MinConfidenceMargin)
	} else {
		switch top.label {
		case "entailment":
			verdict = VerdictEntailed
			reason = fmt.Sprintf("grounded entailment with confidence %.2f", top.prob)
		case "contradiction":
			verdict = VerdictContradicted
			reason = fmt.Sprintf("detected contradiction with confidence %.2f", top.prob)
		case "neutral":
			verdict = VerdictNeutral
			reason = fmt.Sprintf("neutral / ungrounded claim with confidence %.2f", top.prob)
		default:
			verdict = VerdictAmbiguous
			reason = "unrecognized top label"
		}
	}

	return EntailmentContractEvaluation{
		Verdict:          verdict,
		SanitizedPremise: cleanPremise,
		SanitizedClaim:   cleanClaim,
		TopLabel:         top.label,
		TopConfidence:    math.Round(top.prob*1000) / 1000,
		ConfidenceMargin: math.Round(margin*1000) / 1000,
		IsValid:          verdict != VerdictRejected,
		AuditReason:      reason,
	}, nil
}
