package claim

import (
	"testing"
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
