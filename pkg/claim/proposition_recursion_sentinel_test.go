package claim

import (
	"errors"
	"testing"
)

func TestPropositionRecursionSentinel_NominalChain(t *testing.T) {
	sentinel := NewPropositionRecursionSentinel(DefaultPropositionSentinelConfig())

	// Leaf premises
	_ = sentinel.AddProposition(PropositionNode{ID: "p1", Statement: "Mars is the 4th planet"})
	_ = sentinel.AddProposition(PropositionNode{ID: "p2", Statement: "Mars has an atmosphere"})

	// Intermediate claim relying on p1 & p2
	_ = sentinel.AddProposition(PropositionNode{
		ID:        "p3",
		Statement: "Mars can support robotic landers",
		Premises:  []string{"p1", "p2"},
	})

	// Conclusion claim
	_ = sentinel.AddProposition(PropositionNode{
		ID:        "p4",
		Statement: "NASA landed Perseverance on Mars",
		Premises:  []string{"p3"},
	})

	order, err := sentinel.ResolveDependencyChain("p4")
	if err != nil {
		t.Fatalf("unexpected error resolving valid chain: %v", err)
	}

	// Should resolve leaf premises first, then p3, then p4
	if len(order) != 4 {
		t.Fatalf("expected 4 resolved propositions, got %d: %v", len(order), order)
	}
	if order[len(order)-1] != "p4" {
		t.Fatalf("expected target p4 to be last in resolution order, got %s", order[len(order)-1])
	}

	metrics := sentinel.GetMetrics()
	if metrics.TotalPropositions != 4 || metrics.ResolutionsCount != 1 || metrics.CyclesIntercepted != 0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

func TestPropositionRecursionSentinel_DirectCycleIntercept(t *testing.T) {
	sentinel := NewPropositionRecursionSentinel(DefaultPropositionSentinelConfig())

	_ = sentinel.AddProposition(PropositionNode{ID: "cA", Statement: "Claim A", Premises: []string{"cB"}})
	_ = sentinel.AddProposition(PropositionNode{ID: "cB", Statement: "Claim B", Premises: []string{"cA"}})

	_, err := sentinel.ResolveDependencyChain("cA")
	if !errors.Is(err, ErrPropositionCycleDetected) {
		t.Fatalf("expected ErrPropositionCycleDetected, got: %v", err)
	}

	metrics := sentinel.GetMetrics()
	if metrics.CyclesIntercepted != 1 {
		t.Fatalf("expected 1 cycle intercepted, got %d", metrics.CyclesIntercepted)
	}
}

func TestPropositionRecursionSentinel_IndirectMultiHopCycle(t *testing.T) {
	sentinel := NewPropositionRecursionSentinel(DefaultPropositionSentinelConfig())

	// c1 -> c2 -> c3 -> c4 -> c1
	_ = sentinel.AddProposition(PropositionNode{ID: "c1", Premises: []string{"c2"}})
	_ = sentinel.AddProposition(PropositionNode{ID: "c2", Premises: []string{"c3"}})
	_ = sentinel.AddProposition(PropositionNode{ID: "c3", Premises: []string{"c4"}})
	_ = sentinel.AddProposition(PropositionNode{ID: "c4", Premises: []string{"c1"}})

	_, err := sentinel.ResolveDependencyChain("c1")
	if !errors.Is(err, ErrPropositionCycleDetected) {
		t.Fatalf("expected ErrPropositionCycleDetected on 4-hop cycle, got: %v", err)
	}
}

func TestPropositionRecursionSentinel_MaxDepthExceeded(t *testing.T) {
	cfg := PropositionSentinelConfig{MaxRecursionDepth: 3}
	sentinel := NewPropositionRecursionSentinel(cfg)

	// Linear chain of depth 5: d1 -> d2 -> d3 -> d4 -> d5
	_ = sentinel.AddProposition(PropositionNode{ID: "d1", Premises: []string{"d2"}})
	_ = sentinel.AddProposition(PropositionNode{ID: "d2", Premises: []string{"d3"}})
	_ = sentinel.AddProposition(PropositionNode{ID: "d3", Premises: []string{"d4"}})
	_ = sentinel.AddProposition(PropositionNode{ID: "d4", Premises: []string{"d5"}})
	_ = sentinel.AddProposition(PropositionNode{ID: "d5", Premises: nil})

	_, err := sentinel.ResolveDependencyChain("d1")
	if !errors.Is(err, ErrPropositionMaxDepthExceeded) {
		t.Fatalf("expected ErrPropositionMaxDepthExceeded, got: %v", err)
	}

	metrics := sentinel.GetMetrics()
	if metrics.DepthViolations != 1 {
		t.Fatalf("expected 1 depth violation, got %d", metrics.DepthViolations)
	}
}

func TestPropositionRecursionSentinel_SelfReferential(t *testing.T) {
	sentinel := NewPropositionRecursionSentinel(DefaultPropositionSentinelConfig())

	err := sentinel.AddProposition(PropositionNode{
		ID:       "self_claim",
		Premises: []string{"self_claim"},
	})
	if !errors.Is(err, ErrSelfReferentialProposition) {
		t.Fatalf("expected ErrSelfReferentialProposition, got: %v", err)
	}
}
