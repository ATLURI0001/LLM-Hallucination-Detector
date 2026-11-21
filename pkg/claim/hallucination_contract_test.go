package claim

import (
	"testing"
)

func TestSanitizeClaimText(t *testing.T) {
	t.Parallel()

	input := "The **Apollo 11** mission landed on the Moon [1, 2] in July 1969 [citation needed], according to NASA* records."
	expected := "The Apollo 11 mission landed on the Moon in July 1969 , according to NASA records."

	got := SanitizeClaimText(input)
	if got != expected {
		t.Errorf("SanitizeClaimText mismatch:\ngot:      %q\nexpected: %q", got, expected)
	}
}

func TestValidateVerificationRequest_Valid(t *testing.T) {
	t.Parallel()

	spec := VerificationRequestSpec{
		QueryID:   "query-101",
		ClaimText: "Water freezes at 0 degrees Celsius under standard atmospheric pressure.",
		ReferenceContexts: []string{
			"Standard atmospheric freezing point of water is 0 °C (32 °F).",
		},
		ConfidenceThreshold: 0.85,
		CheckMode:           ModeStrict,
	}

	cleaned, err := ValidateVerificationRequest(spec)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if cleaned.QueryID != "query-101" {
		t.Errorf("expected QueryID 'query-101', got %s", cleaned.QueryID)
	}
	if cleaned.ConfidenceThreshold != 0.85 {
		t.Errorf("expected threshold 0.85, got %f", cleaned.ConfidenceThreshold)
	}
	if cleaned.CheckMode != ModeStrict {
		t.Errorf("expected ModeStrict, got %s", cleaned.CheckMode)
	}
	if len(cleaned.SanitizedContexts) != 1 {
		t.Errorf("expected 1 sanitized context, got %d", len(cleaned.SanitizedContexts))
	}
}

func TestValidateVerificationRequest_Defaults(t *testing.T) {
	t.Parallel()

	spec := VerificationRequestSpec{
		QueryID:   "query-default",
		ClaimText: "A valid factual proposition for verification.",
		ReferenceContexts: []string{
			"A reference text with sufficient context length.",
		},
	}

	cleaned, err := ValidateVerificationRequest(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cleaned.ConfidenceThreshold != 0.70 {
		t.Errorf("expected default threshold 0.70, got %f", cleaned.ConfidenceThreshold)
	}
	if cleaned.CheckMode != ModeConservative {
		t.Errorf("expected default mode ModeConservative, got %s", cleaned.CheckMode)
	}
}

func TestValidateVerificationRequest_Errors(t *testing.T) {
	t.Parallel()

	// 1. Invalid QueryID
	_, err := ValidateVerificationRequest(VerificationRequestSpec{
		QueryID:           "?? bad id ??",
		ClaimText:         "Valid claim length.",
		ReferenceContexts: []string{"Valid context."},
	})
	if err != ErrInvalidQueryID {
		t.Errorf("expected ErrInvalidQueryID, got %v", err)
	}

	// 2. Claim too short
	_, err = ValidateVerificationRequest(VerificationRequestSpec{
		QueryID:           "query-1",
		ClaimText:         "tiny",
		ReferenceContexts: []string{"Valid context."},
	})
	if err != ErrClaimTooShort {
		t.Errorf("expected ErrClaimTooShort, got %v", err)
	}

	// 3. No contexts
	_, err = ValidateVerificationRequest(VerificationRequestSpec{
		QueryID:           "query-1",
		ClaimText:         "A valid claim statement.",
		ReferenceContexts: []string{},
	})
	if err != ErrNoReferenceContexts {
		t.Errorf("expected ErrNoReferenceContexts, got %v", err)
	}

	// 4. Invalid threshold
	_, err = ValidateVerificationRequest(VerificationRequestSpec{
		QueryID:             "query-1",
		ClaimText:           "A valid claim statement.",
		ReferenceContexts:   []string{"Valid context text."},
		ConfidenceThreshold: 1.5,
	})
	if err != ErrInvalidThreshold {
		t.Errorf("expected ErrInvalidThreshold, got %v", err)
	}

	// 5. Invalid mode
	_, err = ValidateVerificationRequest(VerificationRequestSpec{
		QueryID:           "query-1",
		ClaimText:         "A valid claim statement.",
		ReferenceContexts: []string{"Valid context text."},
		CheckMode:         "AGGRESSIVE",
	})
	if err != ErrInvalidCheckMode {
		t.Errorf("expected ErrInvalidCheckMode, got %v", err)
	}
}
