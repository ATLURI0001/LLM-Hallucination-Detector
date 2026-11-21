package engine

import (
	"errors"
	"testing"
)

func TestPropositionGraph_TopologicalSort(t *testing.T) {
	g := NewPropositionGraph()

	_ = g.AddProposition(PropositionNode{ID: "p1", Statement: "Premise 1: Sun radiates energy", Confidence: 0.99, IsPremise: true})
	_ = g.AddProposition(PropositionNode{ID: "p2", Statement: "Intermediate: Earth absorbs solar flux", Confidence: 0.95})
	_ = g.AddProposition(PropositionNode{ID: "p3", Statement: "Conclusion: Photosynthesis converts solar flux to sugars", Confidence: 0.92})

	if err := g.AddEntailment("p1", "p2", 1.0); err != nil {
		t.Fatalf("unexpected AddEntailment error: %v", err)
	}
	if err := g.AddEntailment("p2", "p3", 0.95); err != nil {
		t.Fatalf("unexpected AddEntailment error: %v", err)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}
	if order[0] != "p1" || order[1] != "p2" || order[2] != "p3" {
		t.Fatalf("unexpected topological order: %v", order)
	}
}

func TestPropositionGraph_CycleDetection(t *testing.T) {
	g := NewPropositionGraph()

	_ = g.AddProposition(PropositionNode{ID: "c1", Statement: "Claim A", Confidence: 0.8})
	_ = g.AddProposition(PropositionNode{ID: "c2", Statement: "Claim B", Confidence: 0.8})
	_ = g.AddProposition(PropositionNode{ID: "c3", Statement: "Claim C", Confidence: 0.8})

	_ = g.AddEntailment("c1", "c2", 1.0)
	_ = g.AddEntailment("c2", "c3", 1.0)
	_ = g.AddEntailment("c3", "c1", 1.0) // creates cycle: c1 -> c2 -> c3 -> c1

	cycle, err := g.DetectCycles()
	if !errors.Is(err, ErrCycleDetected) {
		t.Fatalf("expected ErrCycleDetected, got err: %v", err)
	}
	if len(cycle) == 0 {
		t.Fatalf("expected non-empty cycle path, got: %v", cycle)
	}

	_, err = g.TopologicalSort()
	if !errors.Is(err, ErrCycleDetected) {
		t.Fatalf("expected TopologicalSort to detect cycle, got %v", err)
	}
}

func TestPropositionGraph_ResolveContradictions(t *testing.T) {
	g := NewPropositionGraph()

	// Premise defeats ungrounded hallucination
	_ = g.AddProposition(PropositionNode{ID: "grounded", Statement: "Revenue was $10M", Confidence: 0.98, IsPremise: true})
	_ = g.AddProposition(PropositionNode{ID: "hallucination", Statement: "Revenue was $100M", Confidence: 0.99, IsPremise: false})

	_ = g.AddConflict("grounded", "hallucination", "Discrepancy in revenue figure")

	pruned := g.ResolveContradictions()
	if pruned != 1 {
		t.Fatalf("expected 1 node pruned, got %d", pruned)
	}

	if g.nodes["grounded"].Pruned {
		t.Errorf("verified premise should not be pruned")
	}
	if !g.nodes["hallucination"].Pruned {
		t.Errorf("ungrounded hallucination should be pruned")
	}

	// Topological sort should ignore pruned nodes
	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed after pruning: %v", err)
	}
	if len(order) != 1 || order[0] != "grounded" {
		t.Fatalf("expected only grounded node in order, got %v", order)
	}
}

func TestPropositionGraph_SelfLoopRejection(t *testing.T) {
	g := NewPropositionGraph()
	_ = g.AddProposition(PropositionNode{ID: "loop", Statement: "Self ref", Confidence: 0.5})

	err := g.AddEntailment("loop", "loop", 1.0)
	if !errors.Is(err, ErrSelfLoopDetected) {
		t.Fatalf("expected ErrSelfLoopDetected on entailment, got %v", err)
	}

	err = g.AddConflict("loop", "loop", "self contradiction")
	if !errors.Is(err, ErrSelfLoopDetected) {
		t.Fatalf("expected ErrSelfLoopDetected on conflict, got %v", err)
	}
}
