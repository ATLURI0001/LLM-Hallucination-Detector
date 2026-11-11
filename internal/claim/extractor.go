package claim

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
)

var (
	// Regex to identify conversational filler and meta-talk from LLMs
	metaTalkRegex = regexp.MustCompile(`(?i)^(sure|certainly|here is|here's|here are|i hope this helps|let me know|feel free|in summary|to summarize|based on the (context|source|provided|document)|according to the context|as an ai|in conclusion|please note|overall|in my opinion|summary of the project)\b`)
	
	// Regex to detect abbreviations that should not break sentences
	abbreviationRegex = regexp.MustCompile(`(?i)\b(mr|mrs|ms|dr|prof|vs|inc|ltd|co|dept|est|approx|fig|e\.g|i\.e|u\.s|u\.k|a\.m|p\.m)\.$`)
	
	// Regex to extract numbers, currencies, dates, and percentages
	numberRegex = regexp.MustCompile(`(?i)(\$|€|£|¥)?\s*\b\d+(?:,\d{3})*(?:\.\d+)?(?:\s*(?:%|percent|k|m|b|bn|million|billion|trillion)\b)?`)
	
	// Regex to extract capitalized proper nouns / named entity candidates
	properNounRegex = regexp.MustCompile(`\b[A-Z][a-z0-9]+(?:\s+[A-Z][a-z0-9]+)*\b`)

	// Compiled once to avoid re-compiling inside sentence loops
	listMarkerRegex = regexp.MustCompile(`^([\*\-•]|\d+[\.\)])\s+`)
)

// Extractor decomposes raw AI-generated text into verifiable claims.
type Extractor struct {
	minClaimLength int
}

// NewExtractor creates a new claim extractor instance.
func NewExtractor(minClaimLength int) *Extractor {
	if minClaimLength <= 0 {
		minClaimLength = 12
	}
	return &Extractor{
		minClaimLength: minClaimLength,
	}
}

// RawSentence represents a sentence slice with text offsets.
type RawSentence struct {
	Text  string
	Start int
	End   int
}

// ExtractClaims extracts atomic claims with offsets from the AI text.
func (e *Extractor) ExtractClaims(text string) []models.Claim {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	rawSentences := e.splitIntoSentences(text)
	var claims []models.Claim
	claimID := 1

	for _, s := range rawSentences {
		cleanText := strings.TrimSpace(s.Text)
		
		// Remove leading bullet marks or numbers: e.g. "- ", "* ", "1. ", "• "
		cleanText = cleanLeadingListMarkers(cleanText)

		// Filter out sentences that are too short
		if len(cleanText) < e.minClaimLength {
			continue
		}

		// Filter out questions and rhetorical statements
		if strings.HasSuffix(cleanText, "?") {
			continue
		}

		// Filter out LLM meta-talk or filler
		if isMetaTalk(cleanText) {
			continue
		}

		// Extract named entities and numeric tokens for downstream verification
		entities := ExtractEntities(cleanText)

		claims = append(claims, models.Claim{
			ID:                claimID,
			Text:              cleanText,
			StartOffset:       s.Start,
			EndOffset:         s.End,
			Status:            models.StatusUnsupported, // Default prior to verification
			HallucinationType: models.HallucinationNone,
			Confidence:        0.0,
			Entities:          entities,
		})
		claimID++
	}

	return claims
}

// splitIntoSentences splits text into sentences while tracking byte offsets.
func (e *Extractor) splitIntoSentences(text string) []RawSentence {
	var sentences []RawSentence
	runes := []rune(text)
	n := len(runes)

	start := 0
	for i := 0; i < n; i++ {
		r := runes[i]

		isTerminator := false
		if r == '.' || r == '!' || r == '?' {
			// Check if next character is space, quote, or end of text
			if i+1 == n || unicode.IsSpace(runes[i+1]) || runes[i+1] == '"' || runes[i+1] == '\'' {
				prefix := string(runes[start : i+1])
				lastWord := getLastWord(prefix)
				if !abbreviationRegex.MatchString(lastWord) {
					isTerminator = true
				}
			}
		} else if r == ':' {
			// Colon followed by newline is a header/preamble terminator
			if i+1 < n && (runes[i+1] == '\n' || runes[i+1] == '\r') {
				isTerminator = true
			}
		} else if r == '\n' {
			// Any newline terminates the current line if non-empty
			isTerminator = true
		}

		if isTerminator && i >= start {
			endIdx := i + 1
			for endIdx < n && (runes[endIdx] == '"' || runes[endIdx] == '\'' || runes[endIdx] == ')' || runes[endIdx] == ']') {
				endIdx++
			}

			subStr := string(runes[start:endIdx])
			trimmed := strings.TrimSpace(subStr)
			if trimmed != "" {
				byteLead := strings.Index(subStr, trimmed)
				runeLead := len([]rune(subStr[:byteLead]))
				actualStart := start + runeLead
				actualEnd := actualStart + len([]rune(trimmed))

				sentences = append(sentences, RawSentence{
					Text:  trimmed,
					Start: actualStart,
					End:   actualEnd,
				})
			}
			start = endIdx
			i = endIdx - 1
		}
	}

	if start < n {
		subStr := string(runes[start:n])
		trimmed := strings.TrimSpace(subStr)
		if trimmed != "" {
			byteLead := strings.Index(subStr, trimmed)
			runeLead := len([]rune(subStr[:byteLead]))
			actualStart := start + runeLead
			actualEnd := actualStart + len([]rune(trimmed))
			sentences = append(sentences, RawSentence{
				Text:  trimmed,
				Start: actualStart,
				End:   actualEnd,
			})
		}
	}

	return sentences
}

func getLastWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func cleanLeadingListMarkers(s string) string {
	s = strings.TrimLeft(s, " \t\r\n")
	return listMarkerRegex.ReplaceAllString(s, "")
}

func isMetaTalk(s string) bool {
	trimmed := strings.TrimSpace(s)
	if strings.HasSuffix(trimmed, ":") {
		return true
	}
	if metaTalkRegex.MatchString(trimmed) {
		words := strings.Fields(trimmed)
		if len(words) <= 15 {
			return true
		}
	}
	return false
}

// ExtractEntities finds numbers, dates, and proper noun candidates.
func ExtractEntities(text string) []string {
	seen := make(map[string]bool)
	var entities []string

	// Extract numbers and metrics
	numMatches := ExtractNumericTokens(text)
	for _, m := range numMatches {
		m = strings.TrimSpace(m)
		if m != "" && !seen[strings.ToLower(m)] {
			seen[strings.ToLower(m)] = true
			entities = append(entities, m)
		}
	}

	// Extract capitalized names/terms
	nameMatches := properNounRegex.FindAllString(text, -1)
	for _, m := range nameMatches {
		m = strings.TrimSpace(m)
		if isCommonWord(m) {
			continue
		}
		if len(m) > 1 && !seen[strings.ToLower(m)] {
			seen[strings.ToLower(m)] = true
			entities = append(entities, m)
		}
	}

	return entities
}

// ExtractNumericTokens specifically isolates all numbers, dates, and currency values.
func ExtractNumericTokens(text string) []string {
	matches := numberRegex.FindAllString(text, -1)
	var result []string
	seen := make(map[string]bool)
	for _, m := range matches {
		cleaned := strings.ToLower(strings.TrimSpace(m))
		if cleaned != "" && !seen[cleaned] {
			seen[cleaned] = true
			result = append(result, strings.TrimSpace(m))
		}
	}
	return result
}

func isCommonWord(w string) bool {
	common := map[string]bool{
		"The": true, "This": true, "These": true, "Those": true, "There": true,
		"In": true, "On": true, "At": true, "For": true, "With": true, "By": true,
		"It": true, "They": true, "We": true, "You": true, "He": true, "She": true,
		"However": true, "Moreover": true, "Furthermore": true, "Therefore": true,
		"Additionally": true, "Consequently": true, "Finally": true, "Although": true,
		"While": true, "Since": true, "Because": true, "According": true, "Overall": true,
	}
	return common[w]
}
