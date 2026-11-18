package models_test

import (
	"testing"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
)

func TestClaim_StatusHelpers(t *testing.T) {
	supported := models.Claim{Status: models.StatusSupported}
	if !supported.IsReliable() {
		t.Fatal("expected supported claim to be reliable")
	}
	if supported.IsHallucination() {
		t.Fatal("expected supported claim not to be hallucination")
	}

	contradicted := models.Claim{Status: models.StatusContradicted}
	if contradicted.IsReliable() {
		t.Fatal("expected contradicted claim not to be reliable")
	}
	if !contradicted.IsHallucination() {
		t.Fatal("expected contradicted claim to be hallucination")
	}

	unsupported := models.Claim{Status: models.StatusUnsupported}
	if unsupported.IsReliable() {
		t.Fatal("expected unsupported claim not to be reliable")
	}
	if !unsupported.IsHallucination() {
		t.Fatal("expected unsupported claim to be hallucination")
	}
}

func TestVerificationOptions_Normalize(t *testing.T) {
	opts := models.DefaultVerificationOptions()
	if opts.Threshold != 0.65 || opts.MinClaimLength != 12 {
		t.Fatalf("unexpected defaults: %+v", opts)
	}

	// Normalization with invalid bounds and un-trimmed strings
	custom := models.VerificationOptions{
		Threshold:      -0.5,
		MinClaimLength: -1,
		LLMAPIKey:      "  test-key  ",
		LLMModel:       "  gemini-flash  ",
		LLMBaseURL:     "  https://api.example.com  ",
	}
	custom.Normalize()
	if custom.Threshold != 0.65 {
		t.Fatalf("expected clamped threshold 0.65, got %f", custom.Threshold)
	}
	if custom.MinClaimLength != 12 {
		t.Fatalf("expected clamped min claim length 12, got %d", custom.MinClaimLength)
	}
	if custom.LLMAPIKey != "test-key" || custom.LLMModel != "gemini-flash" || custom.LLMBaseURL != "https://api.example.com" {
		t.Fatalf("expected trimmed strings, got: %+v", custom)
	}
}

func TestVerificationResult_HasHallucinations(t *testing.T) {
	var nilResult *models.VerificationResult
	if nilResult.HasHallucinations() {
		t.Fatal("expected nil result HasHallucinations() to be false")
	}

	cleanResult := &models.VerificationResult{
		TotalClaims:     5,
		SupportedClaims: 5,
	}
	if cleanResult.HasHallucinations() {
		t.Fatal("expected clean result HasHallucinations() to be false")
	}

	flaggedContradicted := &models.VerificationResult{
		TotalClaims:        5,
		ContradictedClaims: 1,
	}
	if !flaggedContradicted.HasHallucinations() {
		t.Fatal("expected contradicted result to report hallucinations")
	}

	flaggedUnsupported := &models.VerificationResult{
		TotalClaims:       5,
		UnsupportedClaims: 1,
	}
	if !flaggedUnsupported.HasHallucinations() {
		t.Fatal("expected unsupported result to report hallucinations")
	}
}
