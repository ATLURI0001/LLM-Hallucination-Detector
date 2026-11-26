package engine

import (
	"testing"
)

func TestPropositionTraversal_BFS(t *testing.T) {
	profiler := NewPropositionTraversalProfiler()
	graph := NewPropGraph()

	// Build acyclic tree: root -> [c1, c2], c1 -> [c3]
	graph.AddNode("root", "The fiscal budget increased by 4%", []string{"c1", "c2"})
	graph.AddNode("c1", "Revenue grew by 2%", []string{"c3"})
	graph.AddNode("c2", "Expenditure declined by 1%", nil)
	graph.AddNode("c3", "Tax receipts remained stable", nil)

	stats, err := profiler.TraverseBFS(graph, "root")
	if err != nil {
		t.Fatalf("unexpected traversal error: %v", err)
	}

	if stats.VisitedCount != 4 {
		t.Fatalf("expected 4 visited nodes, got %d", stats.VisitedCount)
	}
	if stats.MaxDepth != 2 {
		t.Fatalf("expected max depth 2, got %d", stats.MaxDepth)
	}
	if stats.CycleDetected {
		t.Fatal("unexpected cycle detected in DAG")
	}
	if stats.DurationNs <= 0 {
		t.Fatal("expected positive duration")
	}
}

func TestPropositionTraversal_CycleDetection(t *testing.T) {
	profiler := NewPropositionTraversalProfiler()

	// Acyclic graph
	dag := NewPropGraph()
	dag.AddNode("A", "Statement A", []string{"B"})
	dag.AddNode("B", "Statement B", []string{"C"})
	dag.AddNode("C", "Statement C", nil)

	if profiler.DetectCycles(dag) {
		t.Fatal("expected no cycles in DAG")
	}

	// Cyclic graph: A -> B -> C -> A
	cyclic := NewPropGraph()
	cyclic.AddNode("A", "Statement A", []string{"B"})
	cyclic.AddNode("B", "Statement B", []string{"C"})
	cyclic.AddNode("C", "Statement C", []string{"A"})

	if !profiler.DetectCycles(cyclic) {
		t.Fatal("expected cycle detected in cyclic graph")
	}
}

func TestPropositionTraversal_Benchmark(t *testing.T) {
	profiler := NewPropositionTraversalProfiler()
	graph := NewPropGraph()

	graph.AddNode("n1", "Statement 1", []string{"n2", "n3"})
	graph.AddNode("n2", "Statement 2", []string{"n4"})
	graph.AddNode("n3", "Statement 3", []string{"n4"})
	graph.AddNode("n4", "Statement 4", nil)

	report := profiler.RunBenchmark(graph, []string{"n1"}, 4, 100)

	if report.TotalTraversals != 100 {
		t.Fatalf("expected 100 traversals, got %d", report.TotalTraversals)
	}
	if report.ThroughputTraversalsPerSec <= 0 {
		t.Fatalf("expected positive throughput, got %f", report.ThroughputTraversalsPerSec)
	}
	if report.MeanLatencyNs <= 0 {
		t.Fatalf("expected positive mean latency, got %d", report.MeanLatencyNs)
	}
	if report.P50LatencyNs > report.P99LatencyNs {
		t.Fatalf("p50 %d cannot exceed p99 %d", report.P50LatencyNs, report.P99LatencyNs)
	}
}
