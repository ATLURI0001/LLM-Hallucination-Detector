package report_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/report"
)

func TestFormatSARIF(t *testing.T) {
	res := &models.VerificationResult{
		ID:                 "audit_sarif",
		Timestamp:          time.Now(),
		AnswerText:         "Revenue reached $150 billion.",
		SourceText:         "Quarterly revenue was $90 billion.",
		TotalClaims:        1,
		SupportedClaims:    0,
		ContradictedClaims: 1,
		UnsupportedClaims:  0,
		FaithfulnessScore:  0.0,
		HallucinationScore: 1.0,
		Verdict:            "UNRELIABLE",
		RiskLevel:          "CRITICAL",
		Summary:            "Claim contradicted by source documents.",
		ProcessingTimeMs:   20,
		Claims: []models.Claim{
			{
				ID:                1,
				Text:              "Revenue reached $150 billion.",
				Status:            models.StatusContradicted,
				HallucinationType: models.HallucinationNumeric,
				Confidence:        0.95,
				Explanation:       "Numeric distortion: $150 billion contradicts $90 billion in evidence.",
			},
		},
	}

	sarifJSON, err := report.FormatSARIF(res, "answer.txt")
	if err != nil {
		t.Fatalf("failed to format SARIF: %v", err)
	}

	if !strings.Contains(sarifJSON, "sarif-spec/master/Schemata/sarif-schema-2.1.0.json") && !strings.Contains(sarifJSON, "sarif") {
		t.Errorf("expected SARIF reference in output")
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(sarifJSON), &raw); err != nil {
		t.Fatalf("failed to parse SARIF output: %v", err)
	}

	runs, ok := raw["runs"].([]interface{})
	if !ok || len(runs) == 0 {
		t.Fatalf("expected runs in SARIF report")
	}

	run := runs[0].(map[string]interface{})
	results, ok := run["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results array in SARIF run")
	}

	if len(results) == 0 {
		t.Errorf("expected at least 1 finding for contradicted claim in SARIF results")
	}
}
