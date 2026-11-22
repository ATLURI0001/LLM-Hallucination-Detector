package engine

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// TripleGroundingStatus denotes the grounding status of an entity relationship.
type TripleGroundingStatus string

const (
	TripleStatusGrounded      TripleGroundingStatus = "GROUNDED"
	TripleStatusContradiction TripleGroundingStatus = "CONTRADICTION"
	TripleStatusUngrounded    TripleGroundingStatus = "UNGROUNDED"
)

var (
	ErrEmptyRelationText = errors.New("input text for relation extraction cannot be empty")

	// Curated predicate synonyms for semantic matching
	predicateSynonyms = map[string]string{
		"created":   "invented",
		"invented":  "invented",
		"authored":  "invented",
		"designed":  "invented",
		"founded":   "founded",
		"started":   "founded",
		"located":   "located_in",
		"situated":  "located_in",
		"based":     "located_in",
		"acquired":  "acquired",
		"bought":    "acquired",
		"purchased": "acquired",
	}

	// Regex patterns for extracting simple SVO triples: "<Subject> <verb/predicate> <Object>"
	relationRegex = regexp.MustCompile(`(?i)\b([A-Z][a-zA-Z0-9_\s]{1,30}?)\s+(was\s+)?(invented|created|authored|designed|founded|started|located\s+in|based\s+in|acquired|bought|purchased)\s+(by\s+)?([A-Z0-9][a-zA-Z0-9_\s]{1,40})\b`)
)

// RelationTriple represents a factual (Subject, Predicate, Object) assertion.
type RelationTriple struct {
	Subject   string  `json:"subject"`
	Predicate string  `json:"predicate"`
	Object    string  `json:"object"`
	Confidence float64 `json:"confidence"`
}

// RelationVerificationResult provides per-triple attribution analysis.
type RelationVerificationResult struct {
	Candidate   RelationTriple        `json:"candidate"`
	Status      TripleGroundingStatus `json:"status"`
	Evidence    *RelationTriple       `json:"evidence,omitempty"`
	Explanation string                `json:"explanation"`
}

// DocumentRelationReport summarizes overall relation hallucination scores.
type DocumentRelationReport struct {
	TotalTriples      int                          `json:"total_triples"`
	GroundedCount     int                          `json:"grounded_count"`
	ContradictionCount int                         `json:"contradiction_count"`
	UngroundedCount   int                          `json:"ungrounded_count"`
	HallucinationRate float64                      `json:"hallucination_rate"`
	Results           []RelationVerificationResult `json:"results"`
}

// ExtractRelationTriples parses candidate or reference text to extract SVO relation triples.
func ExtractRelationTriples(text string) ([]RelationTriple, error) {
	cleanText := strings.TrimSpace(text)
	if cleanText == "" {
		return nil, ErrEmptyRelationText
	}

	matches := relationRegex.FindAllStringSubmatch(cleanText, -1)
	var triples []RelationTriple

	for _, m := range matches {
		if len(m) >= 6 {
			sub := strings.TrimSpace(m[1])
			rawPred := strings.TrimSpace(strings.ToLower(m[3]))
			obj := strings.TrimSpace(m[5])

			normPred := normalizePredicate(rawPred)

			triples = append(triples, RelationTriple{
				Subject:    sub,
				Predicate:  normPred,
				Object:     obj,
				Confidence: 0.90,
			})
		}
	}

	return triples, nil
}

// VerifyRelationGrounding compares a candidate triple against reference triples.
func VerifyRelationGrounding(candidate RelationTriple, referenceTriples []RelationTriple) RelationVerificationResult {
	candSub := strings.ToLower(strings.TrimSpace(candidate.Subject))
	candPred := normalizePredicate(candidate.Predicate)
	candObj := strings.ToLower(strings.TrimSpace(candidate.Object))

	var contradictionEvidence *RelationTriple

	for _, ref := range referenceTriples {
		refSub := strings.ToLower(strings.TrimSpace(ref.Subject))
		refPred := normalizePredicate(ref.Predicate)
		refObj := strings.ToLower(strings.TrimSpace(ref.Object))

		// Subject and predicate match
		if (candSub == refSub || strings.Contains(candSub, refSub) || strings.Contains(refSub, candSub)) &&
			(candPred == refPred) {

			// Check object match
			if candObj == refObj || strings.Contains(candObj, refObj) || strings.Contains(refObj, candObj) {
				matchedRef := ref
				return RelationVerificationResult{
					Candidate:   candidate,
					Status:      TripleStatusGrounded,
					Evidence:    &matchedRef,
					Explanation: fmt.Sprintf("Relation grounded by reference: '%s %s %s'", ref.Subject, ref.Predicate, ref.Object),
				}
			}

			// Same subject and predicate, but conflicting object -> CONTRADICTION
			contradictionEvidence = &ref
		}
	}

	if contradictionEvidence != nil {
		return RelationVerificationResult{
			Candidate:   candidate,
			Status:      TripleStatusContradiction,
			Evidence:    contradictionEvidence,
			Explanation: fmt.Sprintf("Contradicts reference fact: reference states '%s %s %s' but candidate claimed '%s'",
				contradictionEvidence.Subject, contradictionEvidence.Predicate, contradictionEvidence.Object, candidate.Object),
		}
	}

	return RelationVerificationResult{
		Candidate:   candidate,
		Status:      TripleStatusUngrounded,
		Explanation: fmt.Sprintf("No supporting evidence found in reference text for '%s %s %s'", candidate.Subject, candidate.Predicate, candidate.Object),
	}
}

// EvaluateDocumentRelations processes candidate triples against reference triples.
func EvaluateDocumentRelations(candidateTriples, referenceTriples []RelationTriple) DocumentRelationReport {
	var results []RelationVerificationResult
	grounded := 0
	contradictions := 0
	ungrounded := 0

	for _, cand := range candidateTriples {
		res := VerifyRelationGrounding(cand, referenceTriples)
		results = append(results, res)

		switch res.Status {
		case TripleStatusGrounded:
			grounded++
		case TripleStatusContradiction:
			contradictions++
		case TripleStatusUngrounded:
			ungrounded++
		}
	}

	total := len(candidateTriples)
	hallucinationRate := 0.0
	if total > 0 {
		hallucinationRate = float64(contradictions+ungrounded) / float64(total)
	}

	return DocumentRelationReport{
		TotalTriples:       total,
		GroundedCount:      grounded,
		ContradictionCount: contradictions,
		UngroundedCount:    ungrounded,
		HallucinationRate:  roundScore(hallucinationRate),
		Results:            results,
	}
}

func normalizePredicate(pred string) string {
	clean := strings.ToLower(strings.TrimSpace(pred))
	clean = strings.TrimSuffix(clean, " in")
	clean = strings.TrimSuffix(clean, " by")
	clean = strings.TrimSpace(clean)
	if canonical, exists := predicateSynonyms[clean]; exists {
		return canonical
	}
	return clean
}

func roundScore(val float64) float64 {
	return float64(int(val*10000+0.5)) / 10000.0
}
