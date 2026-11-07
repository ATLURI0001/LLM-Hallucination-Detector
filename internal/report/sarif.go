package report

import (
	"encoding/json"
	"fmt"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
)

// SarifReport conforms to the OASIS Static Analysis Results Interchange Format (SARIF) v2.1.0.
type SarifReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []SarifRun `json:"runs"`
}

type SarifRun struct {
	Tool    SarifTool     `json:"tool"`
	Results []SarifResult `json:"results"`
}

type SarifTool struct {
	Driver SarifDriver `json:"driver"`
}

type SarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []SarifRule `json:"rules"`
}

type SarifRule struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ShortDescription struct {
		Text string `json:"text"`
	} `json:"shortDescription"`
}

type SarifResult struct {
	RuleID    string `json:"ruleId"`
	Level     string `json:"level"` // error, warning, note
	Message   struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []SarifLocation `json:"locations,omitempty"`
}

type SarifLocation struct {
	PhysicalLocation struct {
		ArtifactLocation struct {
			URI string `json:"uri"`
		} `json:"artifactLocation"`
		Region struct {
			StartLine   int `json:"startLine"`
			StartColumn int `json:"startColumn"`
		} `json:"region"`
	} `json:"physicalLocation"`
}

// FormatSARIF converts verification results into standard SARIF format for CI/CD analysis.
func FormatSARIF(res *models.VerificationResult, filename string) (string, error) {
	if filename == "" {
		filename = "ai_generated_response.txt"
	}

	report := SarifReport{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs: []SarifRun{
			{
				Tool: SarifTool{
					Driver: SarifDriver{
						Name:           "LLM-Hallucination-Detector",
						Version:        "1.0.0",
						InformationURI: "https://github.com/ATLURI0001/llm-hallucination-detector",
						Rules: []SarifRule{
							{
								ID:   "LLM001",
								Name: "DirectContradiction",
								ShortDescription: struct {
									Text string `json:"text"`
								}{Text: "AI assertion contradicts factual statements in the source document."},
							},
							{
								ID:   "LLM002",
								Name: "ExtrinsicHallucination",
								ShortDescription: struct {
									Text string `json:"text"`
								}{Text: "AI assertion introduces unsubstantiated entities or claims."},
							},
							{
								ID:   "LLM003",
								Name: "NumericMismatch",
								ShortDescription: struct {
									Text string `json:"text"`
								}{Text: "AI assertion alters numerical or temporal facts from source document."},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range res.Claims {
		if c.Status == models.StatusSupported {
			continue
		}

		ruleID := "LLM002"
		level := "warning"
		if c.Status == models.StatusContradicted {
			level = "error"
			if c.HallucinationType == models.HallucinationNumeric {
				ruleID = "LLM003"
			} else {
				ruleID = "LLM001"
			}
		}

		resEntry := SarifResult{
			RuleID: ruleID,
			Level:  level,
			Message: struct {
				Text string `json:"text"`
			}{
				Text: fmt.Sprintf("[%s] %s: %s", c.Status, c.Text, c.Explanation),
			},
		}

		loc := SarifLocation{}
		loc.PhysicalLocation.ArtifactLocation.URI = filename
		loc.PhysicalLocation.Region.StartLine = 1
		loc.PhysicalLocation.Region.StartColumn = c.StartOffset + 1
		resEntry.Locations = []SarifLocation{loc}

		report.Runs[0].Results = append(report.Runs[0].Results, resEntry)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
