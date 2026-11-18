package claim

import (
	"testing"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
)

func TestExtractClaims(t *testing.T) {
	extractor := NewExtractor(10)

	aiAnswer := `Sure, here is the summary of the project:
Apollo 11 was launched on July 16, 1969. The mission was commanded by Neil Armstrong.
- In 2024, the budget exceeded $4.5 million.
Dr. Smith confirmed that 98.5% of the data was accurate.
Do you want more details?`

	claims := extractor.ExtractClaims(aiAnswer)

	if len(claims) != 4 {
		t.Fatalf("expected 4 claims, got %d", len(claims))
	}

	expectedPrefixes := []string{
		"Apollo 11 was launched",
		"The mission was commanded",
		"In 2024, the budget",
		"Dr. Smith confirmed that",
	}

	for i, c := range claims {
		if len(c.Text) < len(expectedPrefixes[i]) || c.Text[:len(expectedPrefixes[i])] != expectedPrefixes[i] {
			t.Errorf("claim %d mismatch: got %q, expected prefix %q", i, c.Text, expectedPrefixes[i])
		}
		if len(c.Entities) == 0 {
			t.Errorf("expected entities in claim %d (%s), got none", i, c.Text)
		}
	}
}

func TestExtractNumericTokens(t *testing.T) {
	text := "The revenue rose by 14.2% reaching $500 million in 2023, up from 120k in 2020."
	tokens := ExtractNumericTokens(text)

	expected := []string{"14.2%", "$500 million", "2023", "120k", "2020"}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
}

func TestSplitIntoSentences_UnicodeRuneOffsets(t *testing.T) {
	extractor := NewExtractor(5)
	// Leading multi-byte Unicode whitespace U+2003 (3 UTF-8 bytes, 1 rune) before claim
	text := "\u2003Café revenue rose 20% in Q4."
	sentences := extractor.splitIntoSentences(text)
	if len(sentences) != 1 {
		t.Fatalf("expected 1 sentence, got %d", len(sentences))
	}
	if sentences[0].Start != 1 {
		t.Fatalf("expected rune start offset 1, got %d", sentences[0].Start)
	}
	expectedRunes := len([]rune("Café revenue rose 20% in Q4."))
	if sentences[0].End-sentences[0].Start != expectedRunes {
		t.Fatalf("expected rune length %d, got %d", expectedRunes, sentences[0].End-sentences[0].Start)
	}
}

func TestSplitIntoSentences_SmartQuotesAndClosers(t *testing.T) {
	extractor := NewExtractor(5)
	text := "The analyst stated: “Revenue exceeded $10 million.” Then growth accelerated."
	sentences := extractor.splitIntoSentences(text)
	if len(sentences) != 2 {
		t.Fatalf("expected 2 sentences across smart quote boundary, got %d: %v", len(sentences), sentences)
	}
	if sentences[0].Text != "The analyst stated: “Revenue exceeded $10 million.”" {
		t.Errorf("sentence 0 mismatch, got %q", sentences[0].Text)
	}
	if sentences[1].Text != "Then growth accelerated." {
		t.Errorf("sentence 1 mismatch, got %q", sentences[1].Text)
	}
}

func TestExtractClaims_ExtendedBulletMarkers(t *testing.T) {
	extractor := NewExtractor(5)
	text := `– First milestone completed.
— Second milestone approved.
> Third milestone underway.`
	claims := extractor.ExtractClaims(text)
	if len(claims) != 3 {
		t.Fatalf("expected 3 claims from dash and quote bullets, got %d", len(claims))
	}
	if claims[0].Text != "First milestone completed." {
		t.Errorf("claim 0 mismatch: %q", claims[0].Text)
	}
	if claims[1].Text != "Second milestone approved." {
		t.Errorf("claim 1 mismatch: %q", claims[1].Text)
	}
	if claims[2].Text != "Third milestone underway." {
		t.Errorf("claim 2 mismatch: %q", claims[2].Text)
	}
}

func TestExtractorHelpersAndFilterClaims(t *testing.T) {
	ext := NewExtractor(18)
	if ext.MinClaimLength() != 18 {
		t.Fatalf("expected minClaimLength 18, got %d", ext.MinClaimLength())
	}
	var nilExt *Extractor
	if nilExt.MinClaimLength() != 12 {
		t.Fatalf("expected nil Extractor default 12, got %d", nilExt.MinClaimLength())
	}

	// Meta talk detection
	if !IsMetaTalk("Here is the requested summary:") {
		t.Error("expected meta-talk to be detected")
	}
	if !IsMetaTalk("Certainly! Let me explain:") {
		t.Error("expected conversational opener to be detected")
	}
	if IsMetaTalk("Apollo 11 landed on the Moon in July 1969.") {
		t.Error("factual claim should not be detected as meta-talk")
	}

	// FilterClaims
	claims := []models.Claim{
		{ID: 1, Text: "High confidence claim", Confidence: 0.95},
		{ID: 2, Text: "Low confidence claim", Confidence: 0.40},
		{ID: 3, Text: "Borderline claim", Confidence: 0.65},
	}
	filtered := FilterClaims(claims, 0.65)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered claims, got %d", len(filtered))
	}
	if filtered[0].ID != 1 || filtered[1].ID != 3 {
		t.Errorf("unexpected filtered claim IDs: %v", filtered)
	}

	if FilterClaims(nil, 0.5) != nil {
		t.Error("expected nil for empty claims slice")
	}
}
