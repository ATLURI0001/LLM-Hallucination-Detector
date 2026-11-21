package engine

import (
	"errors"
	"fmt"
	"sort"
)

var (
	ErrCycleDetected    = errors.New("proposition_graph: circular reasoning chain detected")
	ErrNodeNotFound     = errors.New("proposition_graph: proposition node not found")
	ErrSelfLoopDetected = errors.New("proposition_graph: proposition cannot entail itself")
)

// PropositionNode represents a single claim or atomic assertion in a reasoning chain.
type PropositionNode struct {
	ID         string
	Statement  string
	Confidence float64 // 0.0 to 1.0
	Polarity   int     // +1 positive assertion, -1 negated/refuted assertion
	IsPremise  bool    // True if grounded directly in verified source documents
	Pruned     bool    // True if rejected during contradiction resolution
	PruneReason string
}

// PropositionGraph represents an inference graph of entailments and contradictions.
type PropositionGraph struct {
	nodes        map[string]*PropositionNode
	edges        map[string]map[string]float64 // u -> v with entailment strength [0.0, 1.0]
	reverseEdges map[string]map[string]float64 // v -> u
	conflicts    map[string]map[string]string  // u <-> v with conflict rationale
}

// NewPropositionGraph initializes a new inference topology graph.
func NewPropositionGraph() *PropositionGraph {
	return &PropositionGraph{
		nodes:        make(map[string]*PropositionNode),
		edges:        make(map[string]map[string]float64),
		reverseEdges: make(map[string]map[string]float64),
		conflicts:    make(map[string]map[string]string),
	}
}

// AddProposition registers a proposition node in the graph.
func (g *PropositionGraph) AddProposition(node PropositionNode) error {
	if node.ID == "" {
		return errors.New("proposition_graph: node id cannot be empty")
	}
	if node.Confidence < 0.0 || node.Confidence > 1.0 {
		return errors.New("proposition_graph: confidence must be in range [0.0, 1.0]")
	}
	if node.Polarity == 0 {
		node.Polarity = 1
	}

	nCopy := node
	g.nodes[node.ID] = &nCopy
	if _, ok := g.edges[node.ID]; !ok {
		g.edges[node.ID] = make(map[string]float64)
	}
	if _, ok := g.reverseEdges[node.ID]; !ok {
		g.reverseEdges[node.ID] = make(map[string]float64)
	}
	if _, ok := g.conflicts[node.ID]; !ok {
		g.conflicts[node.ID] = make(map[string]string)
	}
	return nil
}

// AddEntailment registers a directed implication from premise u to conclusion v.
func (g *PropositionGraph) AddEntailment(fromID, toID string, weight float64) error {
	if fromID == toID {
		return ErrSelfLoopDetected
	}
	if _, ok := g.nodes[fromID]; !ok {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, fromID)
	}
	if _, ok := g.nodes[toID]; !ok {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, toID)
	}
	if weight <= 0.0 || weight > 1.0 {
		weight = 1.0
	}

	g.edges[fromID][toID] = weight
	g.reverseEdges[toID][fromID] = weight
	return nil
}

// AddConflict registers a bidirectional mutual contradiction between u and v.
func (g *PropositionGraph) AddConflict(idA, idB string, rationale string) error {
	if idA == idB {
		return ErrSelfLoopDetected
	}
	if _, ok := g.nodes[idA]; !ok {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, idA)
	}
	if _, ok := g.nodes[idB]; !ok {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, idB)
	}

	g.conflicts[idA][idB] = rationale
	g.conflicts[idB][idA] = rationale
	return nil
}

// DetectCycles performs 3-color DFS to identify circular reasoning chains.
// Returns a slice of node IDs participating in the first detected cycle.
func (g *PropositionGraph) DetectCycles() ([]string, error) {
	const (
		white = 0 // unvisited
		gray  = 1 // visiting
		black = 2 // visited
	)

	colors := make(map[string]int)
	parent := make(map[string]string)

	var cycleNodes []string

	var dfs func(u string) bool
	dfs = func(u string) bool {
		colors[u] = gray

		// Sort neighbor IDs for deterministic traversal
		neighbors := make([]string, 0, len(g.edges[u]))
		for v := range g.edges[u] {
			if !g.nodes[v].Pruned {
				neighbors = append(neighbors, v)
			}
		}
		sort.Strings(neighbors)

		for _, v := range neighbors {
			if colors[v] == gray {
				// Cycle found: reconstruct path
				cycleNodes = append(cycleNodes, v)
				curr := u
				for curr != v && curr != "" {
					cycleNodes = append(cycleNodes, curr)
					curr = parent[curr]
				}
				cycleNodes = append(cycleNodes, v)
				// Reverse to show natural cycle direction
				for i, j := 0, len(cycleNodes)-1; i < j; i, j = i+1, j-1 {
					cycleNodes[i], cycleNodes[j] = cycleNodes[j], cycleNodes[i]
				}
				return true
			}
			if colors[v] == white {
				parent[v] = u
				if dfs(v) {
					return true
				}
			}
		}

		colors[u] = black
		return false
	}

	allNodeIDs := make([]string, 0, len(g.nodes))
	for id, n := range g.nodes {
		if !n.Pruned {
			allNodeIDs = append(allNodeIDs, id)
		}
	}
	sort.Strings(allNodeIDs)

	for _, id := range allNodeIDs {
		if colors[id] == white {
			if dfs(id) {
				return cycleNodes, ErrCycleDetected
			}
		}
	}

	return nil, nil
}

// TopologicalSort executes Kahn's algorithm returning propositions in order of derivation.
func (g *PropositionGraph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int)
	for id, n := range g.nodes {
		if !n.Pruned {
			inDegree[id] = 0
		}
	}

	for u, neighbors := range g.edges {
		if g.nodes[u].Pruned {
			continue
		}
		for v := range neighbors {
			if !g.nodes[v].Pruned {
				inDegree[v]++
			}
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var sorted []string
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		sorted = append(sorted, u)

		var outNeighbors []string
		for v := range g.edges[u] {
			if !g.nodes[v].Pruned {
				outNeighbors = append(outNeighbors, v)
			}
		}
		sort.Strings(outNeighbors)

		for _, v := range outNeighbors {
			inDegree[v]--
			if inDegree[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	if len(sorted) != len(inDegree) {
		return sorted, ErrCycleDetected
	}

	return sorted, nil
}

// ResolveContradictions evaluates conflicting proposition pairs and prunes weaker / unsupported claims.
func (g *PropositionGraph) ResolveContradictions() (prunedCount int) {
	seenPairs := make(map[string]bool)

	for idA, conflictMap := range g.conflicts {
		nodeA := g.nodes[idA]
		if nodeA.Pruned {
			continue
		}

		for idB, rationale := range conflictMap {
			nodeB := g.nodes[idB]
			if nodeB.Pruned {
				continue
			}

			pairKey := fmt.Sprintf("%s:%s", minStr(idA, idB), maxStr(idA, idB))
			if seenPairs[pairKey] {
				continue
			}
			seenPairs[pairKey] = true

			// Resolution policy:
			// 1. A premise (grounded in source documents) strictly defeats a derived / ungrounded claim.
			// 2. If both are premises or neither is premise, higher confidence score wins.
			// 3. In equal ties, reject both or lower alphanumeric id for determinism.
			if nodeA.IsPremise && !nodeB.IsPremise {
				nodeB.Pruned = true
				nodeB.PruneReason = fmt.Sprintf("Contradicted by verified premise %s: %s", idA, rationale)
				prunedCount++
			} else if !nodeA.IsPremise && nodeB.IsPremise {
				nodeA.Pruned = true
				nodeA.PruneReason = fmt.Sprintf("Contradicted by verified premise %s: %s", idB, rationale)
				prunedCount++
			} else if nodeA.Confidence > nodeB.Confidence {
				nodeB.Pruned = true
				nodeB.PruneReason = fmt.Sprintf("Contradicted with lower confidence (%.2f vs %.2f): %s", nodeB.Confidence, nodeA.Confidence, rationale)
				prunedCount++
			} else if nodeB.Confidence > nodeA.Confidence {
				nodeA.Pruned = true
				nodeA.PruneReason = fmt.Sprintf("Contradicted with lower confidence (%.2f vs %.2f): %s", nodeA.Confidence, nodeB.Confidence, rationale)
				prunedCount++
			} else {
				// Tie: prune nodeB
				nodeB.Pruned = true
				nodeB.PruneReason = fmt.Sprintf("Contradicted under confidence tie with %s: %s", idA, rationale)
				prunedCount++
			}
		}
	}

	return prunedCount
}

func minStr(a, b string) string {
	if a < b {
		return a
	}
	return b
}

func maxStr(a, b string) string {
	if a > b {
		return a
	}
	return b
}
