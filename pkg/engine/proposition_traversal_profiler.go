package engine

import (
	"errors"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"
)

var (
	ErrGraphNil  = errors.New("proposition_traversal: graph cannot be nil")
	ErrRootEmpty = errors.New("proposition_traversal: root node ID cannot be empty")
)

// PropNode represents an atomic proposition statement in a verification DAG.
type PropNode struct {
	ID        string
	Statement string
	Outgoing  []string // IDs of downstream entailed propositions
}

// PropGraph represents a directed graph of propositions and entailment dependencies.
type PropGraph struct {
	Nodes map[string]*PropNode
}

// NewPropGraph initializes an empty proposition graph.
func NewPropGraph() *PropGraph {
	return &PropGraph{
		Nodes: make(map[string]*PropNode),
	}
}

// AddNode adds a proposition node to the graph.
func (g *PropGraph) AddNode(id, statement string, outgoing []string) {
	if outgoing == nil {
		outgoing = []string{}
	}
	g.Nodes[id] = &PropNode{
		ID:        id,
		Statement: statement,
		Outgoing:  outgoing,
	}
}

// TraversalStats captures traversal latency and path metrics.
type TraversalStats struct {
	VisitedCount  int
	MaxDepth      int
	CycleDetected bool
	DurationNs    int64
}

// TraversalBenchmarkReport compiles statistical performance distributions.
type TraversalBenchmarkReport struct {
	TotalTraversals            int
	Concurrency                int
	TotalDuration              time.Duration
	ThroughputTraversalsPerSec float64
	MeanLatencyNs              int64
	P50LatencyNs               int64
	P90LatencyNs               int64
	P99LatencyNs               int64
	CyclesDetected             int
	AllocatedBytes             uint64
	AllocsCount                uint64
}

// PropositionTraversalProfiler executes high-throughput graph traversal profiling.
type PropositionTraversalProfiler struct {
	mu sync.RWMutex
}

// NewPropositionTraversalProfiler constructs a traversal profiler instance.
func NewPropositionTraversalProfiler() *PropositionTraversalProfiler {
	return &PropositionTraversalProfiler{}
}

// TraverseBFS executes a breadth-first search starting from a root node.
func (p *PropositionTraversalProfiler) TraverseBFS(graph *PropGraph, rootID string) (TraversalStats, error) {
	if graph == nil {
		return TraversalStats{}, ErrGraphNil
	}
	if rootID == "" {
		return TraversalStats{}, ErrRootEmpty
	}

	start := time.Now()

	visited := make(map[string]bool)
	queue := []string{rootID}
	visited[rootID] = true

	cycleDetected := false
	maxDepth := 0
	depthMap := make(map[string]int)
	depthMap[rootID] = 0

	for len(queue) > 0 {
		currID := queue[0]
		queue = queue[1:]
		currDepth := depthMap[currID]

		node, exists := graph.Nodes[currID]
		if !exists {
			continue
		}

		for _, neighbor := range node.Outgoing {
			if neighbor == currID || visited[neighbor] {
				// Re-visiting an ancestor indicates a cycle
				cycleDetected = true
				continue
			}
			visited[neighbor] = true
			depthMap[neighbor] = currDepth + 1
			if currDepth+1 > maxDepth {
				maxDepth = currDepth + 1
			}
			queue = append(queue, neighbor)
		}
	}

	elapsedNs := time.Since(start).Nanoseconds()
	if elapsedNs <= 0 {
		elapsedNs = 1
	}

	return TraversalStats{
		VisitedCount:  len(visited),
		MaxDepth:      maxDepth,
		CycleDetected: cycleDetected,
		DurationNs:    elapsedNs,
	}, nil
}

// DetectCycles evaluates whether the proposition graph contains circular entailment dependencies.
func (p *PropositionTraversalProfiler) DetectCycles(graph *PropGraph) bool {
	if graph == nil || len(graph.Nodes) == 0 {
		return false
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycleDFS func(nodeID string) bool
	hasCycleDFS = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		node, exists := graph.Nodes[nodeID]
		if exists {
			for _, neighbor := range node.Outgoing {
				if !visited[neighbor] {
					if hasCycleDFS(neighbor) {
						return true
					}
				} else if recStack[neighbor] {
					return true
				}
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range graph.Nodes {
		if !visited[nodeID] {
			if hasCycleDFS(nodeID) {
				return true
			}
		}
	}

	return false
}

// RunBenchmark executes parallel traversal workloads to measure latency distributions.
func (p *PropositionTraversalProfiler) RunBenchmark(
	graph *PropGraph,
	rootIDs []string,
	concurrency int,
	iterations int,
) TraversalBenchmarkReport {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	if iterations <= 0 {
		iterations = 100
	}
	if len(rootIDs) == 0 {
		rootIDs = []string{"root"}
	}

	totalRuns := len(rootIDs) * iterations

	var mBefore, mAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mBefore)

	startTime := time.Now()

	latencies := make([]int64, 0, totalRuns)
	var latenciesMu sync.Mutex
	cyclesTotal := 0
	var cyclesMu sync.Mutex

	taskChan := make(chan string, totalRuns)
	for i := 0; i < iterations; i++ {
		for _, r := range rootIDs {
			taskChan <- r
		}
	}
	close(taskChan)

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localLatencies := make([]int64, 0, totalRuns/concurrency+1)
			localCycles := 0

			for root := range taskChan {
				stats, err := p.TraverseBFS(graph, root)
				if err == nil {
					localLatencies = append(localLatencies, stats.DurationNs)
					if stats.CycleDetected {
						localCycles++
					}
				}
			}

			latenciesMu.Lock()
			latencies = append(latencies, localLatencies...)
			latenciesMu.Unlock()

			cyclesMu.Lock()
			cyclesTotal += localCycles
			cyclesMu.Unlock()
		}()
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	runtime.ReadMemStats(&mAfter)
	allocatedBytes := mAfter.TotalAlloc - mBefore.TotalAlloc
	allocsCount := mAfter.Mallocs - mBefore.Mallocs

	if len(latencies) == 0 {
		return TraversalBenchmarkReport{
			TotalTraversals: 0,
			Concurrency:     concurrency,
			TotalDuration:   totalDuration,
		}
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var sum int64
	for _, l := range latencies {
		sum += l
	}
	meanNs := sum / int64(len(latencies))

	p50 := latencies[int(math.Floor(float64(len(latencies))*0.50))]
	p90 := latencies[int(math.Floor(float64(len(latencies))*0.90))]
	p99 := latencies[int(math.Floor(float64(len(latencies))*0.99))]

	throughput := float64(len(latencies)) / totalDuration.Seconds()

	return TraversalBenchmarkReport{
		TotalTraversals:            len(latencies),
		Concurrency:                concurrency,
		TotalDuration:              totalDuration,
		ThroughputTraversalsPerSec: math.Round(throughput*100) / 100,
		MeanLatencyNs:              meanNs,
		P50LatencyNs:               p50,
		P90LatencyNs:               p90,
		P99LatencyNs:               p99,
		CyclesDetected:             cyclesTotal,
		AllocatedBytes:             allocatedBytes,
		AllocsCount:                allocsCount,
	}
}
