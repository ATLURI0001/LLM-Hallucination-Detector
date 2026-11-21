package engine

import (
	"testing"
)

func TestExtractTemporalAnchors(t *testing.T) {
	text := "The acquisition occurred on 2021-04-15, following the initial announcement in 2020."
	anchors := ExtractTemporalAnchors(text)

	if len(anchors) < 2 {
		t.Fatalf("expected at least 2 anchors, got %d", len(anchors))
	}

	hasDate := false
	hasYear := false
	for _, a := range anchors {
		if a.Date != nil && a.RawText == "2021-04-15" {
			hasDate = true
		}
		if a.Year == 2020 {
			hasYear = true
		}
	}

	if !hasDate || !hasYear {
		t.Errorf("anchors missing expected date or year: hasDate=%v, hasYear=%v", hasDate, hasYear)
	}
}

func TestVerifyTemporalConsistency_GroundedMatch(t *testing.T) {
	premise := "Apollo 11 landed on the lunar surface in 1969."
	claim := "The historic lunar landing of Apollo 11 took place in 1969."

	res := VerifyTemporalConsistency(premise, claim)
	if res.Status != TemporalGrounded {
		t.Fatalf("expected GROUNDED, got %s", res.Status)
	}
	if res.DeltaYears != 0.0 || res.ConfidenceLoss != 0.0 {
		t.Errorf("expected zero delta and zero confidence loss")
	}
}

func TestVerifyTemporalConsistency_Discrepancy(t *testing.T) {
	premise := "The company was founded in 2012 by two Stanford graduates."
	claim := "The startup was established in 2018."

	res := VerifyTemporalConsistency(premise, claim)
	if res.Status != TemporalContradicted {
		t.Fatalf("expected CONTRADICTED, got %s", res.Status)
	}
	if res.DeltaYears != 6.0 {
		t.Errorf("expected delta 6.0 years, got %.1f", res.DeltaYears)
	}
	if res.ConfidenceLoss < 0.8 {
		t.Errorf("expected significant confidence loss, got %.2f", res.ConfidenceLoss)
	}
}

func TestVerifyTemporalConsistency_Anachronism(t *testing.T) {
	premise := "The protocol was officially published in 2015."
	claim := "Prior to this publication, tests were conducted in 2019."

	res := VerifyTemporalConsistency(premise, claim)
	if res.Status != TemporalAnachronism {
		t.Fatalf("expected ANACHRONISM, got %s", res.Status)
	}
	if res.ConfidenceLoss < 0.9 {
		t.Errorf("expected near-total confidence loss for anachronism, got %.2f", res.ConfidenceLoss)
	}
}

func TestVerifyTemporalConsistency_Ungrounded(t *testing.T) {
	premise := "Photosynthesis generates oxygen in plants."
	claim := "Photosynthesis is an important biological pathway."

	res := VerifyTemporalConsistency(premise, claim)
	if res.Status != TemporalUngrounded {
		t.Fatalf("expected UNGROUNDED on zero temporal anchors, got %s", res.Status)
	}
}
