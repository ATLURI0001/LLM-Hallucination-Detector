package engine

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/claim"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/document"
)

// Detector coordinates claim extraction, document retrieval, and veracity verification.
type Detector struct {
	extractor *claim.Extractor
	heuristic *HeuristicVerifier
	llmJudge  *LLMJudge
	options   VerificationOptions
}

// NewDetector initializes a detection engine with given options.
func NewDetector(opts VerificationOptions) *Detector {
	d := &Detector{
		extractor: claim.NewExtractor(opts.MinClaimLength),
		heuristic: NewHeuristicVerifier(opts.StrictNumericVerification),
		options:   opts,
	}

	if opts.EnableLLMJudge && opts.LLMAPIKey != "" {
		d.llmJudge = NewLLMJudge(opts.LLMAPIKey, opts.LLMModel, opts.LLMBaseURL)
	}

	return d
}

// Verify checks an AI answer against one or more trusted source documents.
func (d *Detector) Verify(ctx context.Context, answerText, sourceText string) (*VerificationResult, error) {
	startTime := time.Now()

	// 1. Index trusted source documents
	idx := document.NewInvertedIndex()
	idx.AddDocument("source_doc_main", sourceText)

	// 2. Extract atomic claims from the AI answer
	extractedClaims := d.extractor.ExtractClaims(answerText)

	if len(extractedClaims) == 0 {
		return &VerificationResult{
			ID:                fmt.Sprintf("vr_%d", time.Now().UnixNano()),
			Timestamp:         time.Now(),
			AnswerText:        answerText,
			SourceText:        sourceText,
			Claims:            nil,
			TotalClaims:       0,
			FaithfulnessScore: 1.0,
			HallucinationScore: 0.0,
			Verdict:           "RELIABLE",
			RiskLevel:         "LOW",
			Summary:           "No substantive factual claims extracted from the response.",
			ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
			OptionsUsed:       d.options,
		}, nil
	}

	// 3. Process claims in parallel
	verifiedClaims := make([]Claim, len(extractedClaims))
	var wg sync.WaitGroup

	for i, c := range extractedClaims {
		wg.Add(1)
		go func(idxNum int, rawClaim Claim) {
			defer wg.Done()

			// Retrieve best corroborating evidence passage
			evidence := idx.FindBestEvidence(rawClaim.Text)

			// Step A: Heuristic Rule-Based Verification
			verdictClaim := d.heuristic.VerifyClaim(rawClaim, evidence, sourceText, d.options.Threshold)

			// Step B: Optional LLM Judge Deep Entailment Check
			if d.llmJudge != nil && d.options.EnableLLMJudge && evidence != nil {
				// Invoke LLM judge for borderline or disputed claims
				if verdictClaim.Status != StatusSupported || verdictClaim.Confidence < 0.85 {
					resp, err := d.llmJudge.EvaluateEntailment(ctx, evidence.Snippet, rawClaim.Text)
					if err == nil && resp != nil {
						verdictClaim.Status = resp.Status
						verdictClaim.HallucinationType = resp.Type
						verdictClaim.Confidence = resp.Confidence
						verdictClaim.Explanation = fmt.Sprintf("[LLM Judge] %s", resp.Explanation)
					}
				}
			}

			verifiedClaims[idxNum] = verdictClaim
		}(i, c)
	}

	wg.Wait()

	// 4. Aggregate metrics and scoring
	var (
		supportedCount  int
		contradictCount int
		unsupportedCount int
		partialCount    int
	)

	for _, c := range verifiedClaims {
		switch c.Status {
		case StatusSupported:
			supportedCount++
		case StatusContradicted:
			contradictCount++
		case StatusUnsupported:
			unsupportedCount++
		case StatusPartiallySupported:
			partialCount++
		}
	}

	total := len(verifiedClaims)
	weightedScore := (float64(supportedCount)*1.0 + float64(partialCount)*0.5) / float64(total)
	// If any contradiction occurs, apply a penalty
	if contradictCount > 0 {
		penalty := float64(contradictCount) * 0.25
		weightedScore = math.Max(0.0, weightedScore-penalty)
	}

	faithfulness := math.Round(weightedScore*100) / 100
	hallucination := math.Round((1.0-faithfulness)*100) / 100

	// Determine Verdict
	verdict := "RELIABLE"
	if contradictCount > 0 || faithfulness < 0.60 {
		verdict = "UNRELIABLE"
	} else if faithfulness < 0.85 || unsupportedCount > 0 || partialCount > 0 {
		verdict = "NEEDS_REVIEW"
	}

	// Determine Risk Level
	riskLevel := "LOW"
	if contradictCount > 1 || faithfulness < 0.40 {
		riskLevel = "CRITICAL"
	} else if contradictCount == 1 || faithfulness < 0.65 {
		riskLevel = "HIGH"
	} else if faithfulness < 0.85 {
		riskLevel = "MEDIUM"
	}

	summary := fmt.Sprintf("Analyzed %d claims: %d supported, %d contradicted, %d unsupported, %d partially supported. Faithfulness: %.0f%%.",
		total, supportedCount, contradictCount, unsupportedCount, partialCount, faithfulness*100)

	return &VerificationResult{
		ID:                      fmt.Sprintf("vr_%d", time.Now().UnixNano()),
		Timestamp:               time.Now(),
		AnswerText:              answerText,
		SourceText:              sourceText,
		Claims:                  verifiedClaims,
		TotalClaims:             total,
		SupportedClaims:         supportedCount,
		ContradictedClaims:      contradictCount,
		UnsupportedClaims:       unsupportedCount,
		PartiallySupportedClaims: partialCount,
		FaithfulnessScore:       faithfulness,
		HallucinationScore:      hallucination,
		Verdict:                 verdict,
		RiskLevel:               riskLevel,
		Summary:                 summary,
		ProcessingTimeMs:        time.Since(startTime).Milliseconds(),
		OptionsUsed:             d.options,
	}, nil
}
