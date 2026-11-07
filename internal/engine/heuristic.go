package engine

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"

	"github.com/user/llm-hallucination-detector/internal/claim"
)

var (
	negationWords = map[string]bool{
		"not": true, "never": true, "no": true, "none": true, "neither": true,
		"unable": true, "failed": true, "denied": true, "prohibited": true,
		"refused": true, "cancelled": true, "rejected": true, "disapproved": true,
		"cannot": true, "can't": true, "won't": true, "isn't": true, "wasn't": true,
		"doesn't": true, "didn't": true, "without": true, "lacks": true,
	}

	antonymPairs = [][]string{
		{"increase", "decrease"},
		{"increased", "decreased"},
		{"rose", "fell"},
		{"rise", "fall"},
		{"growth", "decline"},
		{"profit", "loss"},
		{"success", "failure"},
		{"successful", "unsuccessful"},
		{"approved", "rejected"},
		{"safe", "dangerous"},
		{"effective", "ineffective"},
		{"allowed", "banned"},
		{"accepted", "denied"},
		{"expanded", "contracted"},
		{"higher", "lower"},
		{"gain", "loss"},
	}
)

// HeuristicVerifier executes rule-based NLI, numeric discrepancy, and entity checking.
type HeuristicVerifier struct {
	strictNumeric bool
}

// NewHeuristicVerifier creates a new heuristic verifier.
func NewHeuristicVerifier(strictNumeric bool) *HeuristicVerifier {
	return &HeuristicVerifier{
		strictNumeric: strictNumeric,
	}
}

// VerifyClaim evaluates a single claim against candidate evidence and full source text.
func (h *HeuristicVerifier) VerifyClaim(claim Claim, evidence *Evidence, fullSourceText string, threshold float64) Claim {
	// If no evidence was retrieved at all
	if evidence == nil || evidence.SimilarityScore < 0.05 {
		claim.Status = StatusUnsupported
		claim.HallucinationType = HallucinationExtrinsic
		claim.Confidence = 0.92
		claim.Explanation = "Extrinsic Hallucination: No corroborating evidence found anywhere in the trusted source documents."
		claim.MissingEntities = claim.Entities
		return claim
	}

	claim.BestEvidence = evidence
	evidenceText := evidence.Snippet

	// 1. Check Numeric & Temporal Discrepancies
	numDiscrepancies := h.detectNumericDiscrepancy(claim.Text, evidenceText)
	if len(numDiscrepancies) > 0 {
		claim.NumericDiscrepant = numDiscrepancies
		claim.Status = StatusContradicted
		claim.HallucinationType = HallucinationNumeric
		claim.Confidence = 0.95
		d := numDiscrepancies[0]
		claim.Explanation = fmt.Sprintf("Numeric Distortion: Claim asserts %q but source documents state %q.", d.ClaimToken, d.EvidenceToken)
		return claim
	}

	// 2. Check for Entity Hallucinations (Entities mentioned in claim missing from full source)
	missingEntities := h.findMissingEntities(claim.Entities, fullSourceText)
	if len(missingEntities) > 0 {
		claim.MissingEntities = missingEntities
		claim.Status = StatusUnsupported
		claim.HallucinationType = HallucinationEntity
		claim.Confidence = 0.88
		claim.Explanation = fmt.Sprintf("Entity Hallucination: Claim references entity %q which does not exist in the source text.", strings.Join(missingEntities, ", "))
		return claim
	}

	// 3. Check for Negation & Antonym Inversion (Direct Contradiction)
	isContradiction, contraReason := h.detectPolarityInversion(claim.Text, evidenceText)
	if isContradiction {
		claim.Status = StatusContradicted
		claim.HallucinationType = HallucinationIntrinsic
		claim.Confidence = 0.90
		claim.Explanation = fmt.Sprintf("Direct Intrinsic Contradiction: %s", contraReason)
		return claim
	}

	// 4. Calculate Lexical Information Overlap & Weighted Token Entailment
	coverage, jaccard := h.calculateTokenOverlap(claim.Text, evidenceText)

	// Combine coverage and BM25 normalized score
	combinedConfidence := (coverage*0.65) + (jaccard*0.35)

	if coverage >= threshold || evidence.ExactMatch {
		claim.Status = StatusSupported
		claim.HallucinationType = HallucinationNone
		claim.Confidence = math.Min(1.0, math.Max(combinedConfidence, 0.75))
		claim.Explanation = "Claim is directly supported and grounded by the source document passage."
	} else if coverage >= (threshold - 0.22) {
		claim.Status = StatusPartiallySupported
		claim.HallucinationType = HallucinationContextual
		claim.Confidence = combinedConfidence
		claim.Explanation = "Partially Supported: Core terms align, but specific phrasing, relations, or scope appear extrapolated."
	} else {
		claim.Status = StatusUnsupported
		claim.HallucinationType = HallucinationExtrinsic
		claim.Confidence = math.Max(0.70, 1.0-combinedConfidence)
		claim.Explanation = "Unsupported Claim: Insufficient factual overlap with the retrieved source passage."
	}

	return claim
}

// detectNumericDiscrepancy checks if numbers in the claim diverge from numbers in the evidence.
func (h *HeuristicVerifier) detectNumericDiscrepancy(claimText, evidenceText string) []NumericComparison {
	claimNums := claim.ExtractNumericTokens(claimText)
	if len(claimNums) == 0 {
		return nil
	}

	evidenceNums := claim.ExtractNumericTokens(evidenceText)
	if len(evidenceNums) == 0 {
		return nil
	}

	evidenceSet := make(map[string]bool)
	for _, n := range evidenceNums {
		evidenceSet[normalizeNumberToken(n)] = true
	}

	var discrepancies []NumericComparison
	for _, cn := range claimNums {
		normCN := normalizeNumberToken(cn)
		if normCN == "" {
			continue
		}
		// If the exact number token is not in the evidence snippet
		if !evidenceSet[normCN] {
			// Find nearest evidence number for explanation
			bestEv := evidenceNums[0]
			discrepancies = append(discrepancies, NumericComparison{
				ClaimToken:    strings.TrimSpace(cn),
				EvidenceToken: strings.TrimSpace(bestEv),
				Context:       fmt.Sprintf("Claim has %s vs Evidence %s", cn, bestEv),
				Discrepancy:   fmt.Sprintf("Numerical mismatch: %s != %s", cn, bestEv),
			})
		}
	}

	return discrepancies
}

func normalizeNumberToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "%", "")
	return s
}

// findMissingEntities detects proper nouns in the claim that are never mentioned in the source.
func (h *HeuristicVerifier) findMissingEntities(claimEntities []string, fullSourceText string) []string {
	lowerSource := strings.ToLower(fullSourceText)
	var missing []string

	for _, entity := range claimEntities {
		clean := strings.TrimSpace(entity)
		// Skip purely numeric tokens here (handled by numeric checker)
		isNumeric := true
		for _, r := range clean {
			if !unicode.IsDigit(r) && r != '.' && r != ',' && r != '%' && r != '$' {
				isNumeric = false
				break
			}
		}
		if isNumeric || len(clean) <= 2 {
			continue
		}

		if !strings.Contains(lowerSource, strings.ToLower(clean)) {
			missing = append(missing, clean)
		}
	}

	return missing
}

// detectPolarityInversion identifies negation or antonym conflicts.
func (h *HeuristicVerifier) detectPolarityInversion(claimText, evidenceText string) (bool, string) {
	claimWords := tokenizeWords(claimText)
	evidenceWords := tokenizeWords(evidenceText)

	claimNeg := hasNegation(claimWords)
	evidNeg := hasNegation(evidenceWords)

	// If one is negated and the other is not, AND they share significant content words
	if claimNeg != evidNeg {
		commonContent := countCommonWords(claimWords, evidenceWords)
		if commonContent >= 3 {
			if claimNeg {
				return true, "Claim states a negative assertion where the source document makes an affirmative statement."
			}
			return true, "Source document contains explicit negation contradicting this affirmative claim."
		}
	}

	// Antonym checking
	claimSet := make(map[string]bool)
	for _, w := range claimWords {
		claimSet[w] = true
	}
	evidSet := make(map[string]bool)
	for _, w := range evidenceWords {
		evidSet[w] = true
	}

	for _, pair := range antonymPairs {
		w1, w2 := pair[0], pair[1]
		if (claimSet[w1] && evidSet[w2]) || (claimSet[w2] && evidSet[w1]) {
			return true, fmt.Sprintf("Antonym opposition detected: %s vs %s.", w1, w2)
		}
	}

	return false, ""
}

func tokenizeWords(s string) []string {
	words := regexp.MustCompile(`[\p{L}\p{N}]+`).FindAllString(strings.ToLower(s), -1)
	return words
}

func hasNegation(words []string) bool {
	for _, w := range words {
		if negationWords[w] {
			return true
		}
	}
	return false
}

func countCommonWords(w1, w2 []string) int {
	set := make(map[string]bool)
	for _, w := range w1 {
		if !negationWords[w] && len(w) > 2 {
			set[w] = true
		}
	}
	count := 0
	for _, w := range w2 {
		if set[w] {
			count++
		}
	}
	return count
}

func (h *HeuristicVerifier) calculateTokenOverlap(claimText, evidenceText string) (coverage float64, jaccard float64) {
	cw := tokenizeWords(claimText)
	ew := tokenizeWords(evidenceText)

	if len(cw) == 0 {
		return 0, 0
	}

	stopWords := map[string]bool{
		"the": true, "is": true, "at": true, "which": true, "on": true, "and": true,
		"a": true, "an": true, "in": true, "to": true, "of": true, "for": true, "it": true,
		"was": true, "were": true, "by": true, "with": true, "as": true, "that": true,
	}

	cFiltered := make(map[string]bool)
	for _, w := range cw {
		if !stopWords[w] && len(w) > 1 {
			cFiltered[w] = true
		}
	}

	eFiltered := make(map[string]bool)
	for _, w := range ew {
		if !stopWords[w] && len(w) > 1 {
			eFiltered[w] = true
		}
	}

	if len(cFiltered) == 0 {
		return 0, 0
	}

	matched := 0
	for w := range cFiltered {
		if eFiltered[w] {
			matched++
		}
	}

	coverage = float64(matched) / float64(len(cFiltered))

	unionSize := len(cFiltered)
	for w := range eFiltered {
		if !cFiltered[w] {
			unionSize++
		}
	}

	if unionSize > 0 {
		jaccard = float64(matched) / float64(unionSize)
	}

	return coverage, jaccard
}
