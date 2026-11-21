package engine

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TemporalStatus classifies the temporal relation between claim and premise.
type TemporalStatus string

const (
	TemporalGrounded     TemporalStatus = "GROUNDED"
	TemporalContradicted TemporalStatus = "CONTRADICTED"
	TemporalAnachronism  TemporalStatus = "ANACHRONISM"
	TemporalUngrounded   TemporalStatus = "UNGROUNDED"
)

var (
	yearRegex = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)
	dateRegex = regexp.MustCompile(`\b(19\d{2}|20\d{2})-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])\b`)
)

// TemporalAnchor represents an extracted point in time.
type TemporalAnchor struct {
	RawText string
	Year    int
	Date    *time.Time
}

// TemporalVerificationResult details the chronological consistency audit.
type TemporalVerificationResult struct {
	Status         TemporalStatus `json:"status"`
	PremiseYear    *int           `json:"premise_year,omitempty"`
	ClaimYear      *int           `json:"claim_year,omitempty"`
	DeltaYears     float64        `json:"delta_years"`
	Rationale      string         `json:"rationale"`
	ConfidenceLoss float64        `json:"confidence_loss"` // 0.0 = full confidence, 1.0 = total hallucination
}

// ExtractTemporalAnchors parses years and ISO dates from natural language text.
func ExtractTemporalAnchors(text string) []TemporalAnchor {
	var anchors []TemporalAnchor
	seen := make(map[string]bool)

	// 1. Full ISO dates (YYYY-MM-DD)
	for _, match := range dateRegex.FindAllString(text, -1) {
		if seen[match] {
			continue
		}
		seen[match] = true
		t, err := time.Parse("2006-01-02", match)
		if err == nil {
			anchors = append(anchors, TemporalAnchor{
				RawText: match,
				Year:    t.Year(),
				Date:    &t,
			})
		}
	}

	// 2. Standalone years
	for _, match := range yearRegex.FindAllString(text, -1) {
		if seen[match] {
			continue
		}
		seen[match] = true
		y, err := strconv.Atoi(match)
		if err == nil {
			anchors = append(anchors, TemporalAnchor{
				RawText: match,
				Year:    y,
				Date:    nil,
			})
		}
	}

	return anchors
}

// VerifyTemporalConsistency checks if the claim's temporal references are supported by the premise.
func VerifyTemporalConsistency(premise, claim string) TemporalVerificationResult {
	premiseAnchors := ExtractTemporalAnchors(premise)
	claimAnchors := ExtractTemporalAnchors(claim)

	if len(claimAnchors) == 0 {
		return TemporalVerificationResult{
			Status:         TemporalUngrounded,
			DeltaYears:     0.0,
			Rationale:      "No temporal anchors detected in claim",
			ConfidenceLoss: 0.0,
		}
	}

	if len(premiseAnchors) == 0 {
		return TemporalVerificationResult{
			Status:         TemporalUngrounded,
			DeltaYears:     0.0,
			Rationale:      "Claim contains temporal anchor, but premise lacks temporal context",
			ConfidenceLoss: 0.25,
		}
	}

	// Compare primary anchor (earliest or most prominent)
	pYear := premiseAnchors[0].Year
	cYear := claimAnchors[0].Year
	delta := math.Abs(float64(cYear - pYear))

	// Exact match
	if pYear == cYear {
		return TemporalVerificationResult{
			Status:         TemporalGrounded,
			PremiseYear:    &pYear,
			ClaimYear:      &cYear,
			DeltaYears:     0.0,
			Rationale:      fmt.Sprintf("Temporal anchor %d matches premise", cYear),
			ConfidenceLoss: 0.0,
		}
	}

	// Check if claim asserts an event occurred before premise antecedent (Anachronism)
	if strings.Contains(strings.ToLower(claim), "before") || strings.Contains(strings.ToLower(claim), "prior") {
		if cYear > pYear {
			return TemporalVerificationResult{
				Status:         TemporalAnachronism,
				PremiseYear:    &pYear,
				ClaimYear:      &cYear,
				DeltaYears:     delta,
				Rationale:      fmt.Sprintf("Claim asserts event occurred prior, but year %d postdates premise %d", cYear, pYear),
				ConfidenceLoss: 0.95,
			}
		}
	}

	// Temporal contradiction delta
	confLoss := math.Min(1.0, 0.4+(delta*0.1))
	return TemporalVerificationResult{
		Status:         TemporalContradicted,
		PremiseYear:    &pYear,
		ClaimYear:      &cYear,
		DeltaYears:     delta,
		Rationale:      fmt.Sprintf("Temporal discrepancy of %.0f years between premise (%d) and claim (%d)", delta, pYear, cYear),
		ConfidenceLoss: math.Round(confLoss*100) / 100,
	}
}
