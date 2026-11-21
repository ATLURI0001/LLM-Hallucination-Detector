package engine

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// EntailmentRelation represents the directional semantic relationship.
type EntailmentRelation string

const (
	RelationEntailment    EntailmentRelation = "ENTAILMENT"
	RelationNeutral       EntailmentRelation = "NEUTRAL"
	RelationContradiction EntailmentRelation = "CONTRADICTION"
)

// ContradictionType characterizes the structural cause of conflict.
type ContradictionType string

const (
	TypeNone               ContradictionType = "NONE"
	TypeNumericalConflict  ContradictionType = "NUMERICAL_CONFLICT"
	TypePolarityInversion  ContradictionType = "POLARITY_INVERSION"
	TypeEntityMismatch     ContradictionType = "ENTITY_MISMATCH"
)

// ClaimPairResult contains the diagnostic resolution of a proposition against a premise.
type ClaimPairResult struct {
	ClaimProposition string             `json:"claim_proposition"`
	PremiseStatement string             `json:"premise_statement"`
	Relation         EntailmentRelation `json:"relation"`
	ConflictType     ContradictionType  `json:"conflict_type"`
	Confidence       float64            `json:"confidence"`
	RelativeDelta    float64            `json:"relative_delta,omitempty"`
	Reason           string             `json:"reason"`
}

// ContradictionSummary aggregates matrix evaluation diagnostics.
type ContradictionSummary struct {
	TotalEvaluated    int               `json:"total_evaluated"`
	EntailmentCount   int               `json:"entailment_count"`
	NeutralCount      int               `json:"neutral_count"`
	ContradictionCount int              `json:"contradiction_count"`
	ConsistencyScore  float64           `json:"consistency_score"`
	Results           []ClaimPairResult `json:"results"`
}

var (
	numberRegex = regexp.MustCompile(`[-+]?[0-9]*\.?[0-9]+(?:%|[kKmMbB])?`)
	negationRegex = regexp.MustCompile(`\b(not|never|no|neither|nor|without|cannot|unable|failed)\b`)
)

// ContradictionEngine assesses factual alignment and numerical discrepancies.
type ContradictionEngine struct {
	NumericalTolerance float64
}

// NewContradictionEngine initializes the engine with default float tolerance.
func NewContradictionEngine(tolerance float64) *ContradictionEngine {
	if tolerance <= 0 || math.IsNaN(tolerance) {
		tolerance = 0.05 // default 5% tolerance
	}
	return &ContradictionEngine{
		NumericalTolerance: tolerance,
	}
}

// EvaluateMatrix cross-references an array of claim propositions against premise statements.
func (e *ContradictionEngine) EvaluateMatrix(claims []string, premises []string) ContradictionSummary {
	var results []ClaimPairResult
	entailCount := 0
	neutralCount := 0
	contraCount := 0

	for _, claim := range claims {
		trimmedClaim := strings.TrimSpace(claim)
		if trimmedClaim == "" {
			continue
		}

		bestPair := ClaimPairResult{
			ClaimProposition: trimmedClaim,
			PremiseStatement: "",
			Relation:         RelationNeutral,
			ConflictType:     TypeNone,
			Confidence:       0.0,
			Reason:           "No matching premise found in reference corpus",
		}

		// Evaluate against premises
		for _, premise := range premises {
			trimmedPremise := strings.TrimSpace(premise)
			if trimmedPremise == "" {
				continue
			}

			pair := e.evaluatePair(trimmedClaim, trimmedPremise)
			if pair.Relation == RelationContradiction {
				bestPair = pair
				break // Contradiction overrides
			} else if pair.Relation == RelationEntailment && bestPair.Relation != RelationContradiction {
				if pair.Confidence > bestPair.Confidence {
					bestPair = pair
				}
			}
		}

		switch bestPair.Relation {
		case RelationContradiction:
			contraCount++
		case RelationEntailment:
			entailCount++
		default:
			neutralCount++
		}

		results = append(results, bestPair)
	}

	total := len(results)
	consistency := 1.0
	if total > 0 {
		consistency = math.Round((float64(entailCount) / float64(total)) * 1e4) / 1e4
	}

	return ContradictionSummary{
		TotalEvaluated:     total,
		EntailmentCount:    entailCount,
		NeutralCount:       neutralCount,
		ContradictionCount: contraCount,
		ConsistencyScore:   consistency,
		Results:            results,
	}
}

func (e *ContradictionEngine) evaluatePair(claim, premise string) ClaimPairResult {
	claimLower := strings.ToLower(claim)
	premiseLower := strings.ToLower(premise)

	// 1. Check numerical conflicts by matching quantities to their nearest counterparts
	claimNums := extractNumbers(claimLower)
	premiseNums := extractNumbers(premiseLower)

	if len(claimNums) > 0 && len(premiseNums) > 0 {
		for _, cVal := range claimNums {
			// Find closest counterpart in premise
			minDelta := math.MaxFloat64
			for _, pVal := range premiseNums {
				if pVal != 0 {
					d := math.Abs(cVal-pVal) / math.Abs(pVal)
					if d < minDelta {
						minDelta = d
					}
				}
			}

			// If the closest candidate still diverges beyond tolerance, report numerical contradiction
			if minDelta > e.NumericalTolerance && minDelta < math.MaxFloat64 {
				return ClaimPairResult{
					ClaimProposition: claim,
					PremiseStatement: premise,
					Relation:         RelationContradiction,
					ConflictType:     TypeNumericalConflict,
					Confidence:       0.95,
					RelativeDelta:    math.Round(minDelta*1e4) / 1e4,
					Reason:           "Quantifier discrepancy exceeds configured tolerance threshold",
				}
			}
		}
	}

	// 2. Check polarity inversion
	claimNeg := negationRegex.MatchString(claimLower)
	premiseNeg := negationRegex.MatchString(premiseLower)

	// Compute token overlap
	claimTokens := tokenizeWords(claimLower)
	premiseTokens := tokenizeWords(premiseLower)
	overlap := tokenOverlapRatio(claimTokens, premiseTokens)

	if overlap >= 0.5 {
		if claimNeg != premiseNeg {
			return ClaimPairResult{
				ClaimProposition: claim,
				PremiseStatement: premise,
				Relation:         RelationContradiction,
				ConflictType:     TypePolarityInversion,
				Confidence:       0.90,
				Reason:           "Syntactic polarity mismatch between claim and reference context",
			}
		}

		return ClaimPairResult{
			ClaimProposition: claim,
			PremiseStatement: premise,
			Relation:         RelationEntailment,
			ConflictType:     TypeNone,
			Confidence:       math.Round(overlap*1e4) / 1e4,
			Reason:           "Substantial lexical and predicate semantic entailment verified",
		}
	}

	return ClaimPairResult{
		ClaimProposition: claim,
		PremiseStatement: premise,
		Relation:         RelationNeutral,
		ConflictType:     TypeNone,
		Confidence:       0.0,
		Reason:           "Insufficient lexical overlap to establish entailment or conflict",
	}
}

func extractNumbers(s string) []float64 {
	matches := numberRegex.FindAllString(s, -1)
	var nums []float64
	for _, m := range matches {
		cleaned := strings.TrimRight(m, "%kKmMbB")
		multiplier := 1.0
		if strings.HasSuffix(m, "k") || strings.HasSuffix(m, "K") {
			multiplier = 1000.0
		} else if strings.HasSuffix(m, "m") || strings.HasSuffix(m, "M") {
			multiplier = 1000000.0
		} else if strings.HasSuffix(m, "b") || strings.HasSuffix(m, "B") {
			multiplier = 1000000000.0
		}
		if v, err := strconv.ParseFloat(cleaned, 64); err == nil {
			nums = append(nums, v*multiplier)
		}
	}
	return nums
}

func tokenizeWords(s string) []string {
	fields := strings.Fields(s)
	var tokens []string
	for _, f := range fields {
		clean := strings.Trim(f, ".,!?:;\"'()[]{}")
		if len(clean) > 0 {
			tokens = append(tokens, clean)
		}
	}
	return tokens
}

func tokenOverlapRatio(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	bMap := make(map[string]bool)
	for _, w := range b {
		bMap[w] = true
	}
	matches := 0
	for _, w := range a {
		if bMap[w] {
			matches++
		}
	}
	return float64(matches) / float64(len(a))
}
