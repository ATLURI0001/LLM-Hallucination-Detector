package engine

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
)

var (
	ErrEmptyClaims   = errors.New("cannot construct entailment matrix with empty claims list")
	ErrEmptyPremises = errors.New("cannot construct entailment matrix with empty premises list")
)

// ClaimAttestationStatus indicates the grounding state of an extracted proposition.
type ClaimAttestationStatus string

const (
	StatusVerified     ClaimAttestationStatus = "VERIFIED"
	StatusContradicted ClaimAttestationStatus = "CONTRADICTED"
	StatusUngrounded   ClaimAttestationStatus = "UNGROUNDED"
)

// MatrixCell contains cross-attestation scores between a claim (row i) and premise (col j).
type MatrixCell struct {
	ClaimIndex      int                `json:"claim_index"`
	PremiseIndex    int                `json:"premise_index"`
	Relation        EntailmentRelation `json:"relation"`
	OverlapScore    float64            `json:"overlap_score"`
	EntailmentScore float64            `json:"entailment_score"`
	Reason          string             `json:"reason"`
}

// ClaimAttestationReport details the verdict for an individual claim statement.
type ClaimAttestationReport struct {
	ClaimIndex      int                    `json:"claim_index"`
	ClaimText       string                 `json:"claim_text"`
	Status          ClaimAttestationStatus `json:"status"`
	BestPremiseIdx  int                    `json:"best_premise_idx"`
	BestScore       float64                `json:"best_score"`
	SupportingCells []MatrixCell           `json:"supporting_cells"`
	ConflictingCell *MatrixCell            `json:"conflicting_cell,omitempty"`
}

// CrossSentenceAttestationMatrix stores the full M x N verification grid.
type CrossSentenceAttestationMatrix struct {
	Claims           []string                 `json:"claims"`
	Premises         []string                 `json:"premises"`
	Grid             [][]MatrixCell           `json:"grid"`
	ClaimReports     []ClaimAttestationReport `json:"claim_reports"`
	AttestationIndex float64                  `json:"attestation_index"`
	VerifiedCount    int                      `json:"verified_count"`
	ContradictedCount int                     `json:"contradicted_count"`
	UngroundedCount  int                      `json:"ungrounded_count"`
}

// EntailmentMatrixConfig configures sensitivity thresholds.
type EntailmentMatrixConfig struct {
	MinEntailmentThreshold float64 `json:"min_entailment_threshold"`
	MinWordOverlapRatio    float64 `json:"min_word_overlap_ratio"`
}

// DefaultEntailmentMatrixConfig returns balanced operational thresholds.
func DefaultEntailmentMatrixConfig() EntailmentMatrixConfig {
	return EntailmentMatrixConfig{
		MinEntailmentThreshold: 0.35,
		MinWordOverlapRatio:    0.20,
	}
}

// EntailmentMatrixEngine orchestrates cross-sentence attestation.
type EntailmentMatrixEngine struct {
	mu     sync.RWMutex
	config EntailmentMatrixConfig
}

var wordTokenRegex = regexp.MustCompile(`\b[a-zA-Z0-9_\-]+\b`)

// NewEntailmentMatrixEngine initializes a matrix engine.
func NewEntailmentMatrixEngine(cfg EntailmentMatrixConfig) *EntailmentMatrixEngine {
	if cfg.MinEntailmentThreshold <= 0 || cfg.MinEntailmentThreshold > 1.0 {
		cfg.MinEntailmentThreshold = 0.35
	}
	if cfg.MinWordOverlapRatio <= 0 || cfg.MinWordOverlapRatio > 1.0 {
		cfg.MinWordOverlapRatio = 0.20
	}
	return &EntailmentMatrixEngine{config: cfg}
}

// BuildMatrix constructs an M x N grid cross-referencing claims against premises.
func (e *EntailmentMatrixEngine) BuildMatrix(claims []string, premises []string) (*CrossSentenceAttestationMatrix, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	cleanClaims := cleanSentences(claims)
	cleanPremises := cleanSentences(premises)

	if len(cleanClaims) == 0 {
		return nil, ErrEmptyClaims
	}
	if len(cleanPremises) == 0 {
		return nil, ErrEmptyPremises
	}

	m := len(cleanClaims)
	n := len(cleanPremises)

	grid := make([][]MatrixCell, m)
	claimReports := make([]ClaimAttestationReport, m)

	verifiedCount := 0
	contradictedCount := 0
	ungroundedCount := 0

	for i := 0; i < m; i++ {
		grid[i] = make([]MatrixCell, n)
		claimText := cleanClaims[i]

		var bestCell MatrixCell
		var supporting []MatrixCell
		var conflictCell *MatrixCell
		bestScore := -1.0
		bestPremiseIdx := -1

		for j := 0; j < n; j++ {
			premiseText := cleanPremises[j]
			cell := e.evaluateCell(i, j, claimText, premiseText)
			grid[i][j] = cell

			if cell.Relation == RelationContradiction {
				if conflictCell == nil {
					cCopy := cell
					conflictCell = &cCopy
				}
			} else if cell.Relation == RelationEntailment {
				supporting = append(supporting, cell)
			}

			if cell.EntailmentScore > bestScore {
				bestScore = cell.EntailmentScore
				bestPremiseIdx = j
				bestCell = cell
			}
		}

		// Determine claim verdict
		var status ClaimAttestationStatus
		if conflictCell != nil {
			status = StatusContradicted
			contradictedCount++
		} else if len(supporting) > 0 && bestCell.EntailmentScore >= e.config.MinEntailmentThreshold {
			status = StatusVerified
			verifiedCount++
		} else {
			status = StatusUngrounded
			ungroundedCount++
		}

		claimReports[i] = ClaimAttestationReport{
			ClaimIndex:      i,
			ClaimText:       claimText,
			Status:          status,
			BestPremiseIdx:  bestPremiseIdx,
			BestScore:       math.Round(bestScore*1000) / 1000,
			SupportingCells: supporting,
			ConflictingCell: conflictCell,
		}
	}

	attestationIndex := 0.0
	if m > 0 {
		attestationIndex = math.Round((float64(verifiedCount)/float64(m))*1000) / 1000
	}

	return &CrossSentenceAttestationMatrix{
		Claims:           cleanClaims,
		Premises:         cleanPremises,
		Grid:             grid,
		ClaimReports:     claimReports,
		AttestationIndex: attestationIndex,
		VerifiedCount:    verifiedCount,
		ContradictedCount: contradictedCount,
		UngroundedCount:  ungroundedCount,
	}, nil
}

func (e *EntailmentMatrixEngine) evaluateCell(i, j int, claim, premise string) MatrixCell {
	claimTokens := extractTokenSet(claim)
	premiseTokens := extractTokenSet(premise)

	overlap := computeJaccardOverlap(claimTokens, premiseTokens)

	// Check for direct polarity or numerical contradiction
	contraEngine := NewContradictionEngine(0.05)
	pairRes := contraEngine.evaluatePair(claim, premise)

	relation := RelationNeutral
	entailScore := overlap
	reason := "Insufficient semantic overlap"

	if pairRes.Relation == RelationContradiction {
		relation = RelationContradiction
		entailScore = 0.0
		reason = fmt.Sprintf("Contradiction: %s", pairRes.Reason)
	} else if overlap >= e.config.MinWordOverlapRatio {
		// High lexical overlap without polarity inversion entails
		relation = RelationEntailment
		entailScore = math.Min(1.0, overlap*1.5)
		reason = fmt.Sprintf("Supported by premise with %.1f%% token overlap", overlap*100)
	}

	return MatrixCell{
		ClaimIndex:      i,
		PremiseIndex:    j,
		Relation:        relation,
		OverlapScore:    math.Round(overlap*1000) / 1000,
		EntailmentScore: math.Round(entailScore*1000) / 1000,
		Reason:          reason,
	}
}

func cleanSentences(sentences []string) []string {
	var clean []string
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return clean
}

var commonStopWords = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "in": {}, "on": {}, "at": {},
	"to": {}, "for": {}, "of": {}, "and": {}, "or": {}, "is": {},
	"are": {}, "was": {}, "were": {}, "be": {}, "been": {}, "being": {},
	"have": {}, "has": {}, "had": {}, "do": {}, "does": {}, "did": {},
	"that": {}, "this": {}, "these": {}, "those": {}, "with": {}, "by": {},
	"from": {}, "as": {}, "it": {}, "its": {},
}

func extractTokenSet(text string) map[string]struct{} {
	tokens := make(map[string]struct{})
	matches := wordTokenRegex.FindAllString(strings.ToLower(text), -1)
	for _, m := range matches {
		if len(m) > 1 {
			if _, isStop := commonStopWords[m]; !isStop {
				tokens[m] = struct{}{}
			}
		}
	}
	return tokens
}

func computeJaccardOverlap(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	intersection := 0
	for k := range a {
		if _, exists := b[k]; exists {
			intersection++
		}
	}
	union := len(a)
	for k := range b {
		if _, exists := a[k]; !exists {
			union++
		}
	}
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}
