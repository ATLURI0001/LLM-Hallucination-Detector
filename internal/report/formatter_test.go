package report_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/report"
)

func sampleResult() *models.VerificationResult {
	return &models.VerificationResult{
		ID:                 "audit_123",
		Timestamp:          time.Now(),
		AnswerText:         "Quarterly revenue reached $90 billion.",
		SourceText:         "Quarterly revenue was $90 billion.",
		TotalClaims:        1,
		SupportedClaims:    1,
		ContradictedClaims: 0,
		UnsupportedClaims:  0,
		FaithfulnessScore:  1.0,
		HallucinationScore: 0.0,
		Verdict:            "RELIABLE",
		RiskLevel:          "LOW",
		Summary:            "All claims fully grounded.",
		ProcessingTimeMs:   15,
		Claims: []models.Claim{
			{
				ID:                1,
				Text:              "Quarterly revenue reached $90 billion.",
				Status:            models.StatusSupported,
				HallucinationType: models.HallucinationNone,
				Confidence:        0.98,
				Explanation:       "Exact numeric and semantic alignment.",
				BestEvidence: &models.Evidence{
					DocumentID:      "doc_1",
					Snippet:         "Quarterly revenue was $90 billion.",
					SimilarityScore: 1.0,
					ExactMatch:      true,
				},
			},
		},
	}
}

func TestFormatTerminal(t *testing.T) {
	res := sampleResult()

	t.Run("WithColor", func(t *testing.T) {
		out := report.FormatTerminal(res, false)
		if !strings.Contains(out, "LLM HALLUCINATION & FACT VERIFICATION REPORT") {
			t.Errorf("expected header in terminal output")
		}
		if !strings.Contains(out, "RELIABLE") {
			t.Errorf("expected verdict 'RELIABLE' in output")
		}
	})

	t.Run("NoColor", func(t *testing.T) {
		out := report.FormatTerminal(res, true)
		if strings.Contains(out, "\033[") {
			t.Errorf("expected no ANSI color codes in no-color mode")
		}
		if !strings.Contains(out, "Quarterly revenue reached $90 billion.") {
			t.Errorf("expected claim text in output")
		}
	})

	t.Run("UnreliableVerdict", func(t *testing.T) {
		unreliable := sampleResult()
		unreliable.Verdict = "UNRELIABLE"
		unreliable.RiskLevel = "CRITICAL"
		unreliable.FaithfulnessScore = 0.2
		unreliable.HallucinationScore = 0.8
		unreliable.Claims[0].Status = models.StatusContradicted
		unreliable.Claims[0].HallucinationType = models.HallucinationNumeric

		out := report.FormatTerminal(unreliable, true)
		if !strings.Contains(out, "UNRELIABLE") {
			t.Errorf("expected 'UNRELIABLE' in output")
		}
	})
}

func TestFormatJSON(t *testing.T) {
	res := sampleResult()

	out, err := report.FormatJSON(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded models.VerificationResult
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if decoded.Verdict != "RELIABLE" {
		t.Errorf("expected verdict 'RELIABLE', got %q", decoded.Verdict)
	}
	if len(decoded.Claims) != 1 {
		t.Errorf("expected 1 claim in decoded JSON, got %d", len(decoded.Claims))
	}
}

func TestFormatMarkdown(t *testing.T) {
	res := sampleResult()
	out := report.FormatMarkdown(res)

	if !strings.Contains(out, "LLM Hallucination & Fact Verification Report") {
		t.Errorf("expected Markdown header in output")
	}
	if !strings.Contains(out, "Faithfulness Index") {
		t.Errorf("expected Faithfulness Index metric in table")
	}
	if !strings.Contains(out, "Quarterly revenue reached $90 billion.") {
		t.Errorf("expected claim text in details table")
	}
}
