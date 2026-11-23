package engine

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// SimilarityScore represents an evaluated match between a claim and a reference sentence.
type SimilarityScore struct {
	ClaimIndex     int
	ReferenceIndex int
	CosineSim      float64
	JaccardSim     float64
	OverlapTokens  int
}

// ProfilerProfileReport contains benchmark and profiling telemetry across claim evaluation.
type ProfilerProfileReport struct {
	TotalClaims      int
	TotalReferences  int
	PairEvaluations  int
	Duration         time.Duration
	OpsPerSecond     float64
	MeanCosine       float64
	MaxCosine        float64
	TopMatches       []SimilarityScore
}

// ClaimSimilarityProfiler provides optimized token vectorization and similarity computation.
type ClaimSimilarityProfiler struct {
	mu sync.RWMutex
}

// NewClaimSimilarityProfiler constructs a new similarity profiler.
func NewClaimSimilarityProfiler() *ClaimSimilarityProfiler {
	return &ClaimSimilarityProfiler{}
}

// TokenizeAndFilter decomposes text into lowercase alphanumeric token slices.
func TokenizeAndFilter(text string) []string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		wLower := strings.ToLower(w)
		if len(wLower) > 1 { // Filter single-character noise
			tokens = append(tokens, wLower)
		}
	}
	return tokens
}

// SparseVector converts token slices into term-frequency maps.
func SparseVector(tokens []string) map[string]float64 {
	vec := make(map[string]float64, len(tokens))
	for _, t := range tokens {
		vec[t]++
	}
	return vec
}

// ComputeCosineSimilarity evaluates the normalized dot product of two sparse vectors.
func ComputeCosineSimilarity(vecA, vecB map[string]float64) float64 {
	if len(vecA) == 0 || len(vecB) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for k, vA := range vecA {
		normA += vA * vA
		if vB, ok := vecB[k]; ok {
			dotProduct += vA * vB
		}
	}
	for _, vB := range vecB {
		normB += vB * vB
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ComputeJaccardSimilarity computes token set intersection over union.
func ComputeJaccardSimilarity(tokensA, tokensB []string) (float64, int) {
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0, 0
	}

	setA := make(map[string]struct{}, len(tokensA))
	for _, t := range tokensA {
		setA[t] = struct{}{}
	}

	overlap := 0
	union := make(map[string]struct{}, len(tokensA)+len(tokensB))
	for k := range setA {
		union[k] = struct{}{}
	}

	for _, t := range tokensB {
		union[t] = struct{}{}
		if _, ok := setA[t]; ok {
			overlap++
		}
	}

	if len(union) == 0 {
		return 0.0, 0
	}

	return float64(overlap) / float64(len(union)), overlap
}

// ProfileSimilarity performs batch similarity evaluation across claims and references.
func (p *ClaimSimilarityProfiler) ProfileSimilarity(claims []string, references []string, topK int) ProfilerProfileReport {
	start := time.Now()

	claimTokens := make([][]string, len(claims))
	claimVectors := make([]map[string]float64, len(claims))
	for i, c := range claims {
		claimTokens[i] = TokenizeAndFilter(c)
		claimVectors[i] = SparseVector(claimTokens[i])
	}

	refTokens := make([][]string, len(references))
	refVectors := make([]map[string]float64, len(references))
	for j, r := range references {
		refTokens[j] = TokenizeAndFilter(r)
		refVectors[j] = SparseVector(refTokens[j])
	}

	var scores []SimilarityScore
	var sumCosine float64
	var maxCosine float64
	evalCount := len(claims) * len(references)

	for i := range claims {
		for j := range references {
			cos := ComputeCosineSimilarity(claimVectors[i], refVectors[j])
			jacc, overlap := ComputeJaccardSimilarity(claimTokens[i], refTokens[j])

			sumCosine += cos
			if cos > maxCosine {
				maxCosine = cos
			}

			if cos > 0.1 || jacc > 0.1 {
				scores = append(scores, SimilarityScore{
					ClaimIndex:     i,
					ReferenceIndex: j,
					CosineSim:      math.Round(cos*1000) / 1000,
					JaccardSim:     math.Round(jacc*1000) / 1000,
					OverlapTokens:  overlap,
				})
			}
		}
	}

	sort.Slice(scores, func(a, b int) bool {
		return scores[a].CosineSim > scores[b].CosineSim
	})

	if topK > 0 && len(scores) > topK {
		scores = scores[:topK]
	}

	elapsed := time.Since(start)
	opsPerSec := 0.0
	if elapsed > 0 {
		opsPerSec = float64(evalCount) / elapsed.Seconds()
	}

	meanCosine := 0.0
	if evalCount > 0 {
		meanCosine = sumCosine / float64(evalCount)
	}

	return ProfilerProfileReport{
		TotalClaims:     len(claims),
		TotalReferences: len(references),
		PairEvaluations: evalCount,
		Duration:        elapsed,
		OpsPerSecond:    opsPerSec,
		MeanCosine:      math.Round(meanCosine*1000) / 1000,
		MaxCosine:       math.Round(maxCosine*1000) / 1000,
		TopMatches:      scores,
	}
}
