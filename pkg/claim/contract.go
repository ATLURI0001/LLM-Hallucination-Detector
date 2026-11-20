package claim

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrEmptyClaim      = errors.New("claim text cannot be empty or solely whitespace")
	ErrClaimTooLong    = errors.New("claim text exceeds maximum allowed byte length")
	ErrTooFewTokens    = errors.New("claim proposition does not meet minimum word count requirement")
	ErrPropositionsNil = errors.New("cannot extract propositions from empty text")
)

const (
	MaxClaimLengthBytes = 4096
	MinPropositionWords = 3
)

// ANSI escape sequence regex
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Common conversational filler prefixes that dilute proposition grounding
var conversationalFillers = []string{
	"as an ai,",
	"as an ai language model,",
	"sure, here is",
	"certainly! here is",
	"based on my knowledge,",
	"i believe that",
	"in my opinion,",
}

// SanitizedClaim represents a clean, validated claim ready for hallucination evaluation.
type SanitizedClaim struct {
	OriginalText string
	CleanedText  string
	WordCount    int
	HasAnsi      bool
	HasControl   bool
}

// SanitizeText removes ANSI escape sequences, control characters, and normalizes Unicode whitespace.
func SanitizeText(raw string) (string, bool, bool) {
	hadAnsi := ansiRegex.MatchString(raw)
	cleaned := ansiRegex.ReplaceAllString(raw, "")

	var sb strings.Builder
	hadControl := false

	for _, r := range cleaned {
		// Strip BOM and zero-width spaces
		if r == '\uFEFF' || r == '\u200B' || r == '\u200C' || r == '\u200D' {
			continue
		}
		// Convert non-breaking spaces to standard space
		if r == '\u00A0' || r == '\u202F' {
			sb.WriteRune(' ')
			continue
		}
		// Strip non-printable ASCII control characters (0-31) except newline and tab
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			hadControl = true
			sb.WriteRune(' ')
			continue
		}
		sb.WriteRune(r)
	}

	// Normalize spaces
	normalized := strings.Join(strings.Fields(sb.String()), " ")
	return normalized, hadAnsi, hadControl
}

// StripConversationalFillers trims redundant LLM preambles from propositions.
func StripConversationalFillers(text string) string {
	lower := strings.ToLower(text)
	for _, filler := range conversationalFillers {
		if strings.HasPrefix(lower, filler) {
			trimmed := text[len(filler):]
			return strings.TrimSpace(trimmed)
		}
	}
	return text
}

// ValidateClaimContract enforces structural and size constraints on raw claim text.
func ValidateClaimContract(rawClaim string) (*SanitizedClaim, error) {
	if len(rawClaim) > MaxClaimLengthBytes {
		return nil, ErrClaimTooLong
	}

	cleaned, hadAnsi, hadControl := SanitizeText(rawClaim)
	cleaned = StripConversationalFillers(cleaned)

	if len(cleaned) == 0 {
		return nil, ErrEmptyClaim
	}

	words := strings.Fields(cleaned)
	if len(words) < MinPropositionWords {
		return nil, ErrTooFewTokens
	}

	return &SanitizedClaim{
		OriginalText: rawClaim,
		CleanedText:  cleaned,
		WordCount:    len(words),
		HasAnsi:      hadAnsi,
		HasControl:   hadControl,
	}, nil
}

// ExtractAtomicPropositions decomposes a paragraph into discrete, attestable propositions.
func ExtractAtomicPropositions(paragraph string) ([]string, error) {
	cleaned, _, _ := SanitizeText(paragraph)
	if len(cleaned) == 0 {
		return nil, ErrPropositionsNil
	}

	// Split by terminal punctuation (. ? !) or semicolons
	delimiters := regexp.MustCompile(`[.!?;]+|\n+`)
	segments := delimiters.Split(cleaned, -1)

	var propositions []string
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		seg = StripConversationalFillers(seg)
		if len(seg) == 0 {
			continue
		}

		words := strings.Fields(seg)
		if len(words) >= MinPropositionWords {
			propositions = append(propositions, seg)
		}
	}

	if len(propositions) == 0 {
		return nil, ErrTooFewTokens
	}

	return propositions, nil
}
