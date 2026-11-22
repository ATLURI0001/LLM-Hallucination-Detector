package engine

import (
	"testing"
)

func TestExtractRelationTriples_Extraction(t *testing.T) {
	text := "Python was created by Guido van Rossum. Linux was founded by Linus Torvalds."
	triples, err := ExtractRelationTriples(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(triples) < 2 {
		t.Fatalf("expected at least 2 triples, got %d", len(triples))
	}

	// Verify Python extraction
	foundPython := false
	for _, tr := range triples {
		if tr.Subject == "Python" && tr.Predicate == "invented" && tr.Object == "Guido van Rossum" {
			foundPython = true
			break
		}
	}
	if !foundPython {
		t.Fatalf("expected Python created triple, got: %+v", triples)
	}
}

func TestVerifyRelationGrounding_GroundedAndSynonyms(t *testing.T) {
	refTriples := []RelationTriple{
		{Subject: "Python", Predicate: "created", Object: "Guido van Rossum"},
	}

	// Candidate uses synonym "invented"
	cand := RelationTriple{Subject: "Python", Predicate: "invented", Object: "Guido van Rossum"}
	res := VerifyRelationGrounding(cand, refTriples)

	if res.Status != TripleStatusGrounded {
		t.Fatalf("expected GROUNDED status, got: %s (reason: %s)", res.Status, res.Explanation)
	}
	if res.Evidence == nil {
		t.Fatal("expected evidence to be attached")
	}
}

func TestVerifyRelationGrounding_Contradiction(t *testing.T) {
	refTriples := []RelationTriple{
		{Subject: "Python", Predicate: "created", Object: "Guido van Rossum"},
	}

	// Hallucinated author
	cand := RelationTriple{Subject: "Python", Predicate: "created", Object: "Alan Turing"}
	res := VerifyRelationGrounding(cand, refTriples)

	if res.Status != TripleStatusContradiction {
		t.Fatalf("expected CONTRADICTION status, got: %s", res.Status)
	}
	if res.Evidence == nil || res.Evidence.Object != "Guido van Rossum" {
		t.Fatalf("unexpected evidence: %+v", res.Evidence)
	}
}

func TestVerifyRelationGrounding_Ungrounded(t *testing.T) {
	refTriples := []RelationTriple{
		{Subject: "Linux", Predicate: "started", Object: "Linus Torvalds"},
	}

	// Completely unrelated fact
	cand := RelationTriple{Subject: "Ruby", Predicate: "designed", Object: "Matz"}
	res := VerifyRelationGrounding(cand, refTriples)

	if res.Status != TripleStatusUngrounded {
		t.Fatalf("expected UNGROUNDED status, got: %s", res.Status)
	}
}

func TestEvaluateDocumentRelations_SummaryScores(t *testing.T) {
	refTriples := []RelationTriple{
		{Subject: "Python", Predicate: "created", Object: "Guido van Rossum"},
		{Subject: "Linux", Predicate: "started", Object: "Linus Torvalds"},
	}

	candTriples := []RelationTriple{
		{Subject: "Python", Predicate: "invented", Object: "Guido van Rossum"}, // GROUNDED
		{Subject: "Linux", Predicate: "started", Object: "Bill Gates"},         // CONTRADICTION
		{Subject: "Rust", Predicate: "designed", Object: "Graydon Hoare"},      // UNGROUNDED
	}

	report := EvaluateDocumentRelations(candTriples, refTriples)

	if report.TotalTriples != 3 {
		t.Fatalf("expected 3 total triples, got %d", report.TotalTriples)
	}
	if report.GroundedCount != 1 {
		t.Errorf("expected 1 grounded, got %d", report.GroundedCount)
	}
	if report.ContradictionCount != 1 {
		t.Errorf("expected 1 contradiction, got %d", report.ContradictionCount)
	}
	if report.UngroundedCount != 1 {
		t.Errorf("expected 1 ungrounded, got %d", report.UngroundedCount)
	}
	// Hallucination rate = (1 + 1) / 3 = 0.6667
	if report.HallucinationRate < 0.66 || report.HallucinationRate > 0.67 {
		t.Errorf("expected hallucination rate ~0.6667, got %f", report.HallucinationRate)
	}
}

func TestExtractRelationTriples_EmptyInput(t *testing.T) {
	_, err := ExtractRelationTriples("   ")
	if err != ErrEmptyRelationText {
		t.Fatalf("expected ErrEmptyRelationText, got: %v", err)
	}
}
