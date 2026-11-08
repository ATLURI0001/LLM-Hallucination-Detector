package engine

import (
	"context"
	"testing"
)

func BenchmarkDetectorVerify_Small(b *testing.B) {
	opts := DefaultVerificationOptions()
	detector := NewDetector(opts)
	ctx := context.Background()

	sourceText := "Apple Inc. announced quarterly revenue of $90 billion, representing an 8% increase year-over-year."
	aiAnswer := "Apple reported quarterly revenue of $90 billion."

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := detector.Verify(ctx, aiAnswer, sourceText)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkDetectorVerify_Medium(b *testing.B) {
	opts := DefaultVerificationOptions()
	detector := NewDetector(opts)
	ctx := context.Background()

	sourceText := `QuantumCorp announced Q3 2024 revenue of $420 million, representing a 15% increase year-over-year.
The research division successfully deployed 1,200 qubit processors in Munich, Germany.
CEO Alice Walker confirmed that operations in France were halted due to regulatory hurdles.
The company plans to invest $50 million in European data centers by 2025.`

	aiAnswer := `QuantumCorp announced Q3 2024 revenue of $420 million.
The research division deployed 1,200 qubit processors in Munich.
Operations in France were temporarily suspended by CEO Alice Walker.
QuantumCorp will invest $50 million in European infrastructure.`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := detector.Verify(ctx, aiAnswer, sourceText)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkHeuristicEntailment(b *testing.B) {
	opts := DefaultVerificationOptions()
	verifier := NewHeuristicVerifier(opts.StrictNumericVerification)

	claimObj := Claim{
		Text: "Alphabet reported Q3 revenue of $88.27 billion.",
	}
	evidenceObj := &Evidence{
		Snippet:         "Alphabet reported $88.27 billion in revenue for Q3 2024, beating consensus estimates.",
		SimilarityScore: 0.92,
	}
	fullSource := evidenceObj.Snippet

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = verifier.VerifyClaim(claimObj, evidenceObj, fullSource, opts.Threshold)
	}
}

func BenchmarkTokenOverlap(b *testing.B) {
	verifier := NewHeuristicVerifier(true)
	claimText := "Apple reported Q3 quarterly revenue of $90 billion with solid growth."
	evidenceText := "Apple Inc. announced quarterly revenue of $90 billion, demonstrating solid continuous growth across divisions."

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = verifier.calculateTokenOverlap(claimText, evidenceText)
	}
}

func BenchmarkNumericDiscrepancy(b *testing.B) {
	verifier := NewHeuristicVerifier(true)
	claimText := "The aircraft reached 45,000 feet at Mach 2.4 across 1,800 miles."
	evidenceText := "The prototype aircraft reached 42,000 feet at Mach 2.4 across 1,800 miles."

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = verifier.detectNumericDiscrepancy(claimText, evidenceText)
	}
}
