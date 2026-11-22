package engine

import (
	"errors"
	"sync"
	"testing"
)

func TestEntailmentMatrix_FullyVerified(t *testing.T) {
	engine := NewEntailmentMatrixEngine(DefaultEntailmentMatrixConfig())

	claims := []string{
		"The Apollo 11 mission landed humans on the Moon in July 1969.",
		"Neil Armstrong was the commander of Apollo 11.",
	}

	premises := []string{
		"Apollo 11 was the American spaceflight that first landed humans on the Moon in July 1969.",
		"Commander Neil Armstrong and lunar module pilot Buzz Aldrin formed the crew.",
	}

	matrix, err := engine.BuildMatrix(claims, premises)
	if err != nil {
		t.Fatalf("unexpected error building matrix: %v", err)
	}

	if len(matrix.Grid) != 2 || len(matrix.Grid[0]) != 2 {
		t.Fatalf("expected 2x2 grid, got %dx%d", len(matrix.Grid), len(matrix.Grid[0]))
	}

	if matrix.VerifiedCount != 2 {
		t.Errorf("expected 2 verified claims, got %d", matrix.VerifiedCount)
	}
	if matrix.ContradictedCount != 0 {
		t.Errorf("expected 0 contradicted claims, got %d", matrix.ContradictedCount)
	}
	if matrix.UngroundedCount != 0 {
		t.Errorf("expected 0 ungrounded claims, got %d", matrix.UngroundedCount)
	}
	if matrix.AttestationIndex != 1.0 {
		t.Errorf("expected attestation index 1.0, got %f", matrix.AttestationIndex)
	}
}

func TestEntailmentMatrix_ContradictionDetection(t *testing.T) {
	engine := NewEntailmentMatrixEngine(DefaultEntailmentMatrixConfig())

	claims := []string{
		"The company revenue was 50 million dollars in 2023.", // Contradicts premise (40 million)
		"The headquarters is located in Tokyo.",               // Verified
	}

	premises := []string{
		"In 2023, the reported annual revenue reached 40 million dollars.",
		"The global headquarters is located in central Tokyo.",
	}

	matrix, err := engine.BuildMatrix(claims, premises)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if matrix.ContradictedCount != 1 {
		t.Errorf("expected 1 contradicted claim, got %d", matrix.ContradictedCount)
	}
	if matrix.VerifiedCount != 1 {
		t.Errorf("expected 1 verified claim, got %d", matrix.VerifiedCount)
	}
	if matrix.ClaimReports[0].Status != StatusContradicted {
		t.Errorf("expected claim 0 to be CONTRADICTED, got %s", matrix.ClaimReports[0].Status)
	}
	if matrix.ClaimReports[0].ConflictingCell == nil {
		t.Errorf("expected non-nil ConflictingCell for contradicted claim")
	}
}

func TestEntailmentMatrix_UngroundedHallucination(t *testing.T) {
	engine := NewEntailmentMatrixEngine(DefaultEntailmentMatrixConfig())

	claims := []string{
		"The project was written entirely in Rust.", // Ungrounded hallucination
	}

	premises := []string{
		"The company was founded in 2018 in Seattle.",
		"Their first product was an analytics dashboard for ecommerce.",
	}

	matrix, err := engine.BuildMatrix(claims, premises)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if matrix.UngroundedCount != 1 {
		t.Errorf("expected 1 ungrounded claim, got %d", matrix.UngroundedCount)
	}
	if matrix.ClaimReports[0].Status != StatusUngrounded {
		t.Errorf("expected UNGROUNDED status, got %s", matrix.ClaimReports[0].Status)
	}
	if matrix.AttestationIndex != 0.0 {
		t.Errorf("expected 0.0 attestation index, got %f", matrix.AttestationIndex)
	}
}

func TestEntailmentMatrix_ValidationErrors(t *testing.T) {
	engine := NewEntailmentMatrixEngine(DefaultEntailmentMatrixConfig())

	_, err := engine.BuildMatrix([]string{}, []string{"Premise 1"})
	if !errors.Is(err, ErrEmptyClaims) {
		t.Errorf("expected ErrEmptyClaims, got %v", err)
	}

	_, err = engine.BuildMatrix([]string{"Claim 1"}, []string{"  "})
	if !errors.Is(err, ErrEmptyPremises) {
		t.Errorf("expected ErrEmptyPremises, got %v", err)
	}
}

func TestEntailmentMatrix_Concurrency(t *testing.T) {
	engine := NewEntailmentMatrixEngine(DefaultEntailmentMatrixConfig())
	var wg sync.WaitGroup

	claims := []string{"Model accuracy reached 95% on benchmark."}
	premises := []string{"The evaluation showed 95% accuracy on standard benchmark."}

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := engine.BuildMatrix(claims, premises)
			if err != nil || m.VerifiedCount != 1 {
				t.Errorf("concurrent build matrix failure: %v", err)
			}
		}()
	}

	wg.Wait()
}
