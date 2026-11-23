package engine

import (
	"fmt"
	"testing"
)

func TestTokenizeAndFilter(t *testing.T) {
	text := "Apollo 11 landed on the Moon in July 1969!"
	tokens := TokenizeAndFilter(text)
	expected := []string{"apollo", "11", "landed", "on", "the", "moon", "in", "july", "1969"}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, exp := range expected {
		if tokens[i] != exp {
			t.Errorf("token %d: expected %s, got %s", i, exp, tokens[i])
		}
	}
}

func TestComputeCosineSimilarity_IdenticalAndOrthogonal(t *testing.T) {
	vecA := map[string]float64{"apple": 1.0, "banana": 2.0}
	vecB := map[string]float64{"apple": 1.0, "banana": 2.0}
	vecC := map[string]float64{"orange": 3.0}

	simIdentical := ComputeCosineSimilarity(vecA, vecB)
	if simIdentical < 0.999 {
		t.Errorf("expected ~1.0 for identical vectors, got %f", simIdentical)
	}

	simOrthogonal := ComputeCosineSimilarity(vecA, vecC)
	if simOrthogonal != 0.0 {
		t.Errorf("expected 0.0 for orthogonal vectors, got %f", simOrthogonal)
	}

	simEmpty := ComputeCosineSimilarity(nil, vecA)
	if simEmpty != 0.0 {
		t.Errorf("expected 0.0 for nil vector, got %f", simEmpty)
	}
}

func TestComputeJaccardSimilarity(t *testing.T) {
	tokensA := []string{"apollo", "11", "moon", "mission"}
	tokensB := []string{"apollo", "moon", "orbit"}

	sim, overlap := ComputeJaccardSimilarity(tokensA, tokensB)
	if overlap != 2 {
		t.Errorf("expected 2 overlapping tokens, got %d", overlap)
	}
	// Union: apollo, 11, moon, mission, orbit (5 tokens). Jaccard = 2/5 = 0.4
	if sim < 0.39 || sim > 0.41 {
		t.Errorf("expected ~0.4 jaccard similarity, got %f", sim)
	}
}

func TestClaimSimilarityProfiler_ProfileSimilarity(t *testing.T) {
	profiler := NewClaimSimilarityProfiler()

	claims := []string{
		"Neil Armstrong was the first person to walk on the Moon.",
		"The Great Wall of China is visible from deep outer space.",
	}
	references := []string{
		"Neil Armstrong stepped onto the lunar surface in 1969 during Apollo 11.",
		"Low Earth orbit satellites can photograph large civil engineering structures.",
		"The Apollo program completed six crewed landings on the Moon.",
	}

	report := profiler.ProfileSimilarity(claims, references, 3)

	if report.TotalClaims != 2 || report.TotalReferences != 3 {
		t.Errorf("unexpected counts: claims=%d refs=%d", report.TotalClaims, report.TotalReferences)
	}
	if report.PairEvaluations != 6 {
		t.Errorf("expected 6 pair evaluations, got %d", report.PairEvaluations)
	}
	if report.MaxCosine <= 0.0 {
		t.Errorf("expected positive max cosine similarity, got %f", report.MaxCosine)
	}
	if len(report.TopMatches) == 0 {
		t.Errorf("expected top matches to be populated")
	}
}

func BenchmarkClaimSimilarityProfiler_Evaluation(b *testing.B) {
	profiler := NewClaimSimilarityProfiler()

	claims := make([]string, 20)
	for i := range claims {
		claims[i] = fmt.Sprintf("Candidate claim entity %d asserts factual relation in domain %d", i, i%5)
	}

	references := make([]string, 50)
	for j := range references {
		references[j] = fmt.Sprintf("Reference document passage %d validates ground truth in domain %d", j, j%5)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.ProfileSimilarity(claims, references, 10)
	}
}
