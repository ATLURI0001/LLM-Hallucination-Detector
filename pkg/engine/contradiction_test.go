package engine

import (
	"testing"
)

func TestContradictionEngine_NumericalConflict(t *testing.T) {
	engine := NewContradictionEngine(0.05) // 5% tolerance

	claims := []string{"Q3 net revenue reached 150M dollars across European markets"}
	premises := []string{"Q3 net revenue reached 120M dollars across European markets"}

	res := engine.EvaluateMatrix(claims, premises)
	if res.TotalEvaluated != 1 {
		t.Fatalf("expected 1 evaluated, got %d", res.TotalEvaluated)
	}
	if res.ContradictionCount != 1 {
		t.Fatalf("expected 1 contradiction, got %d", res.ContradictionCount)
	}

	result := res.Results[0]
	if result.Relation != RelationContradiction {
		t.Errorf("expected CONTRADICTION, got %s", result.Relation)
	}
	if result.ConflictType != TypeNumericalConflict {
		t.Errorf("expected NUMERICAL_CONFLICT, got %s", result.ConflictType)
	}
	if result.RelativeDelta != 0.25 {
		t.Errorf("expected relative delta 0.25, got %f", result.RelativeDelta)
	}
}

func TestContradictionEngine_PolarityInversion(t *testing.T) {
	engine := NewContradictionEngine(0.05)

	claims := []string{"The clinical drug candidate was not approved by regulatory bodies"}
	premises := []string{"The clinical drug candidate was approved by regulatory bodies"}

	res := engine.EvaluateMatrix(claims, premises)
	if res.ContradictionCount != 1 {
		t.Fatalf("expected 1 contradiction, got %d", res.ContradictionCount)
	}

	result := res.Results[0]
	if result.ConflictType != TypePolarityInversion {
		t.Errorf("expected POLARITY_INVERSION, got %s", result.ConflictType)
	}
}

func TestContradictionEngine_EntailmentAndNeutral(t *testing.T) {
	engine := NewContradictionEngine(0.05)

	claims := []string{
		"The system architecture relies on distributed message queues",
		"Unrelated proposition regarding deep ocean marine biology",
	}
	premises := []string{
		"The system architecture relies on distributed message queues for high throughput",
	}

	res := engine.EvaluateMatrix(claims, premises)
	if res.TotalEvaluated != 2 {
		t.Fatalf("expected 2 evaluated, got %d", res.TotalEvaluated)
	}
	if res.EntailmentCount != 1 {
		t.Errorf("expected 1 entailment, got %d", res.EntailmentCount)
	}
	if res.NeutralCount != 1 {
		t.Errorf("expected 1 neutral, got %d", res.NeutralCount)
	}
	if res.ConsistencyScore != 0.5 {
		t.Errorf("expected consistency 0.5, got %f", res.ConsistencyScore)
	}
}
