package claim

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeText(t *testing.T) {
	raw := "\x1b[31;1mCrucial claim\x1b[0m with zero-width\u200B and non-breaking\u00A0space and control\x07char."
	cleaned, hadAnsi, hadControl := SanitizeText(raw)

	if !hadAnsi {
		t.Fatal("expected hadAnsi to be true")
	}
	if !hadControl {
		t.Fatal("expected hadControl to be true")
	}
	if strings.Contains(cleaned, "\x1b") {
		t.Fatal("cleaned text must not contain ANSI escape bytes")
	}
	if strings.ContainsRune(cleaned, '\u200B') {
		t.Fatal("cleaned text must not contain zero-width space")
	}
	if strings.ContainsRune(cleaned, '\x07') {
		t.Fatal("cleaned text must not contain control characters")
	}
	expected := "Crucial claim with zero-width and non-breaking space and control char."
	if cleaned != expected {
		t.Fatalf("expected %q, got %q", expected, cleaned)
	}
}

func TestStripConversationalFillers(t *testing.T) {
	in := "As an AI, the speed of light in vacuum is constant."
	out := StripConversationalFillers(in)
	expected := "the speed of light in vacuum is constant."
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}

	clean := "Direct claim with no preamble."
	if StripConversationalFillers(clean) != clean {
		t.Fatal("should not alter text without fillers")
	}
}

func TestValidateClaimContract(t *testing.T) {
	// Exceeds max length
	huge := strings.Repeat("word ", 1000) // 5000 bytes
	_, errHuge := ValidateClaimContract(huge)
	if !errors.Is(errHuge, ErrClaimTooLong) {
		t.Fatalf("expected ErrClaimTooLong, got %v", errHuge)
	}

	// Empty
	_, errEmpty := ValidateClaimContract("   \t  \n  ")
	if !errors.Is(errEmpty, ErrEmptyClaim) {
		t.Fatalf("expected ErrEmptyClaim, got %v", errEmpty)
	}

	// Too few tokens (< 3 words)
	_, errFew := ValidateClaimContract("Single.")
	if !errors.Is(errFew, ErrTooFewTokens) {
		t.Fatalf("expected ErrTooFewTokens, got %v", errFew)
	}

	// Valid claim
	validText := "  \x1b[32mThe Apollo 11 mission landed on the Moon in July 1969.\x1b[0m  "
	sc, err := ValidateClaimContract(validText)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if sc.WordCount != 11 {
		t.Fatalf("expected word count 11, got %d", sc.WordCount)
	}
	if !sc.HasAnsi {
		t.Fatal("expected HasAnsi to be true")
	}
	if sc.CleanedText != "The Apollo 11 mission landed on the Moon in July 1969." {
		t.Fatalf("unexpected cleaned text: %q", sc.CleanedText)
	}
}

func TestExtractAtomicPropositions(t *testing.T) {
	paragraph := "Apollo 11 landed on the Moon in 1969. Neil Armstrong took the first step! Ok. Buzz Aldrin accompanied him shortly afterwards."
	propositions, err := ExtractAtomicPropositions(paragraph)
	if err != nil {
		t.Fatalf("unexpected extraction error: %v", err)
	}

	// "Ok." is < 3 words and should be discarded
	if len(propositions) != 3 {
		t.Fatalf("expected 3 atomic propositions, got %d: %+v", len(propositions), propositions)
	}

	if propositions[0] != "Apollo 11 landed on the Moon in 1969" {
		t.Fatalf("unexpected proposition 0: %q", propositions[0])
	}
	if propositions[1] != "Neil Armstrong took the first step" {
		t.Fatalf("unexpected proposition 1: %q", propositions[1])
	}
	if propositions[2] != "Buzz Aldrin accompanied him shortly afterwards" {
		t.Fatalf("unexpected proposition 2: %q", propositions[2])
	}
}
