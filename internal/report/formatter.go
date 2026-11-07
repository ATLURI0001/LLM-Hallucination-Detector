package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/user/llm-hallucination-detector/internal/models"
)

// ANSI color escape codes for terminal formatting
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// FormatTerminal creates a beautiful colorized terminal report.
func FormatTerminal(res *models.VerificationResult, noColor bool) string {
	cRed := colorRed
	cGreen := colorGreen
	cYellow := colorYellow
	cBold := colorBold
	cDim := colorDim
	cReset := colorReset
	cCyan := colorCyan

	if noColor {
		cRed, cGreen, cYellow, cBold, cDim, cReset, cCyan = "", "", "", "", "", "", ""
	}

	var sb strings.Builder

	sb.WriteString("\n" + cBold + cCyan + "================================================================================" + cReset + "\n")
	sb.WriteString(cBold + "                LLM HALLUCINATION & FACT VERIFICATION REPORT                    " + cReset + "\n")
	sb.WriteString(cCyan + "================================================================================" + cReset + "\n\n")

	// Verdict Banner
	verdictColor := cGreen
	if res.Verdict == "UNRELIABLE" {
		verdictColor = cRed
	} else if res.Verdict == "NEEDS_REVIEW" {
		verdictColor = cYellow
	}

	sb.WriteString(fmt.Sprintf("%sOVERALL VERDICT:%s %s%s%s  |  %sRISK LEVEL:%s %s  |  %sPROCESSING TIME:%s %dms\n\n",
		cBold, cReset, verdictColor+cBold, res.Verdict, cReset,
		cBold, cReset, res.RiskLevel,
		cBold, cReset, res.ProcessingTimeMs))

	// Metrics Summary Card
	sb.WriteString(cBold + "METRICS SUMMARY:\n" + cReset)
	sb.WriteString(fmt.Sprintf("  * Faithfulness Score : %s%.1f%%%s\n", verdictColor+cBold, res.FaithfulnessScore*100, cReset))
	sb.WriteString(fmt.Sprintf("  * Hallucination Rate : %s%.1f%%%s\n", cRed+cBold, res.HallucinationScore*100, cReset))
	sb.WriteString(fmt.Sprintf("  * Total Claims       : %d\n", res.TotalClaims))
	sb.WriteString(fmt.Sprintf("    - %sSupported%s          : %d\n", cGreen, cReset, res.SupportedClaims))
	sb.WriteString(fmt.Sprintf("    - %sContradicted%s        : %d\n", cRed, cReset, res.ContradictedClaims))
	sb.WriteString(fmt.Sprintf("    - %sUnsupported%s         : %d\n", cYellow, cReset, res.UnsupportedClaims))
	sb.WriteString(fmt.Sprintf("    - %sPartially Supported%s : %d\n", cDim, cReset, res.PartiallySupportedClaims))
	sb.WriteString("\n")

	// Detailed Claims Breakdown
	sb.WriteString(cBold + "CLAIMS VERIFICATION BREAKDOWN:\n" + cReset)
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, c := range res.Claims {
		badgeColor := cGreen
		switch c.Status {
		case models.StatusContradicted:
			badgeColor = cRed
		case models.StatusUnsupported:
			badgeColor = cYellow
		case models.StatusPartiallySupported:
			badgeColor = cCyan
		}

		sb.WriteString(fmt.Sprintf("[%s#%d %s%s%s] %s%s%s (Confidence: %.0f%%)\n",
			cBold, c.ID, badgeColor+cBold, c.Status, cReset,
			cBold, c.Text, cReset,
			c.Confidence*100))

		if c.HallucinationType != models.HallucinationNone {
			sb.WriteString(fmt.Sprintf("    %s! Failure Type:%s %s%s%s\n", cRed+cBold, cReset, cRed, c.HallucinationType, cReset))
		}

		sb.WriteString(fmt.Sprintf("    %sExplanation:%s  %s\n", cDim, cReset, c.Explanation))

		for _, nd := range c.NumericDiscrepant {
			sb.WriteString(fmt.Sprintf("    %sNumeric Alert:%s Claim has '%s' vs Source '%s'\n", cRed, cReset, nd.ClaimToken, nd.EvidenceToken))
		}

		if len(c.MissingEntities) > 0 {
			sb.WriteString(fmt.Sprintf("    %sMissing Entity:%s Not found in source: %s\n", cRed, cReset, strings.Join(c.MissingEntities, ", ")))
		}

		if c.BestEvidence != nil {
			snippet := strings.TrimSpace(c.BestEvidence.Snippet)
			if len(snippet) > 160 {
				snippet = snippet[:157] + "..."
			}
			sb.WriteString(fmt.Sprintf("    %sSource Grounding:%s \"%s\"\n", cDim, cReset, snippet))
		}

		sb.WriteString(strings.Repeat("-", 80) + "\n")
	}

	return sb.String()
}

// FormatMarkdown generates a GitHub-flavored Markdown audit report.
func FormatMarkdown(res *models.VerificationResult) string {
	var sb strings.Builder

	sb.WriteString("# 🛡️ LLM Hallucination & Fact Verification Report\n\n")
	sb.WriteString(fmt.Sprintf("> **Generated:** %s | **Audit ID:** `%s` | **Processing Time:** %dms\n\n",
		res.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC"), res.ID, res.ProcessingTimeMs))

	// Status callout
	statusEmoji := "✅"
	calloutType := "NOTE"
	if res.Verdict == "UNRELIABLE" {
		statusEmoji = "🚨"
		calloutType = "CAUTION"
	} else if res.Verdict == "NEEDS_REVIEW" {
		statusEmoji = "⚠️"
		calloutType = "WARNING"
	}

	sb.WriteString(fmt.Sprintf("> [!%s]\n", calloutType))
	sb.WriteString(fmt.Sprintf("> **Verdict: %s %s** (Risk Level: `%s`)\n", statusEmoji, res.Verdict, res.RiskLevel))
	sb.WriteString(fmt.Sprintf("> %s\n\n", res.Summary))

	// Score Cards Table
	sb.WriteString("## 📊 Verification Metrics\n\n")
	sb.WriteString("| Metric | Value | Threshold Status |\n")
	sb.WriteString("| :--- | :--- | :--- |\n")
	sb.WriteString(fmt.Sprintf("| **Faithfulness Index** | **%.1f%%** | %s |\n", res.FaithfulnessScore*100, scoreBadge(res.FaithfulnessScore)))
	sb.WriteString(fmt.Sprintf("| **Hallucination Rate** | **%.1f%%** | %s |\n", res.HallucinationScore*100, hallucinationBadge(res.HallucinationScore)))
	sb.WriteString(fmt.Sprintf("| **Total Claims Extracted** | %d | Factual assertions evaluated |\n", res.TotalClaims))
	sb.WriteString(fmt.Sprintf("| **Supported Claims** | %d | Fully grounded in source |\n", res.SupportedClaims))
	sb.WriteString(fmt.Sprintf("| **Contradicted Claims** | %d | Contradicts source assertions |\n", res.ContradictedClaims))
	sb.WriteString(fmt.Sprintf("| **Unsupported Claims** | %d | Extrinsic hallucinations |\n", res.UnsupportedClaims))
	sb.WriteString(fmt.Sprintf("| **Partially Supported** | %d | Minor ambiguity or drift |\n\n", res.PartiallySupportedClaims))

	// Claims Detail Table
	sb.WriteString("## 🔍 Claims Veracity Analysis\n\n")
	sb.WriteString("| ID | Claim Proposition | Status | Type | Confidence | Grounding Evidence |\n")
	sb.WriteString("| :-: | :--- | :--- | :--- | :-: | :--- |\n")

	for _, c := range res.Claims {
		badge := "🟢 Supported"
		switch c.Status {
		case models.StatusContradicted:
			badge = "🔴 Contradicted"
		case models.StatusUnsupported:
			badge = "🟡 Unsupported"
		case models.StatusPartiallySupported:
			badge = "🔵 Partial"
		}

		evidenceText := "*No grounding found*"
		if c.BestEvidence != nil {
			snip := strings.ReplaceAll(c.BestEvidence.Snippet, "|", "\\|")
			snip = strings.ReplaceAll(snip, "\n", " ")
			if len(snip) > 120 {
				snip = snip[:117] + "..."
			}
			evidenceText = fmt.Sprintf("\"%s\"", snip)
		}

		claimText := strings.ReplaceAll(c.Text, "|", "\\|")

		sb.WriteString(fmt.Sprintf("| #%d | %s | **%s** | `%s` | %.0f%% | %s |\n",
			c.ID, claimText, badge, c.HallucinationType, c.Confidence*100, evidenceText))
	}

	sb.WriteString("\n## 🔬 Evidence Citations & Rationales\n\n")
	for _, c := range res.Claims {
		if c.Status != models.StatusSupported {
			sb.WriteString(fmt.Sprintf("### Claim #%d: *\"%s\"*\n", c.ID, c.Text))
			sb.WriteString(fmt.Sprintf("- **Status:** `%s` (%s)\n", c.Status, c.HallucinationType))
			sb.WriteString(fmt.Sprintf("- **Analysis:** %s\n", c.Explanation))
			if len(c.NumericDiscrepant) > 0 {
				for _, nd := range c.NumericDiscrepant {
					sb.WriteString(fmt.Sprintf("- **Numeric Discrepancy:** Claim stated `%s`, source reported `%s`\n", nd.ClaimToken, nd.EvidenceToken))
				}
			}
			if len(c.MissingEntities) > 0 {
				sb.WriteString(fmt.Sprintf("- **Missing Entities:** `%s` not found in source text\n", strings.Join(c.MissingEntities, ", ")))
			}
			if c.BestEvidence != nil {
				sb.WriteString(fmt.Sprintf("- **Best Source Passage:** > %s\n", c.BestEvidence.Snippet))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// FormatJSON produces pretty-printed JSON output.
func FormatJSON(res *models.VerificationResult) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func scoreBadge(score float64) string {
	if score >= 0.90 {
		return "🟢 EXCELLENT"
	} else if score >= 0.70 {
		return "🟡 ACCEPTABLE"
	}
	return "🔴 UNACCEPTABLE"
}

func hallucinationBadge(rate float64) string {
	if rate <= 0.10 {
		return "🟢 LOW RISK"
	} else if rate <= 0.30 {
		return "🟡 MODERATE RISK"
	}
	return "🔴 HIGH RISK"
}
