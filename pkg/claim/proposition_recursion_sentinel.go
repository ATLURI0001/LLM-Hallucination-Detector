package claim

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrPropositionCycleDetected    = errors.New("proposition dependency cycle detected")
	ErrPropositionMaxDepthExceeded = errors.New("proposition recursion depth exceeded maximum limit")
	ErrPropositionNotFound         = errors.New("proposition not found in dependency graph")
	ErrSelfReferentialProposition  = errors.New("self-referential proposition dependency")
)

type TruthStatus string

const (
	TruthEntailed     TruthStatus = "ENTAILED"
	TruthContradicted TruthStatus = "CONTRADICTED"
	TruthNeutral      TruthStatus = "NEUTRAL"
	TruthUnknown      TruthStatus = "UNKNOWN"
)

type PropositionNode struct {
	ID          string      `json:"id"`
	Statement   string      `json:"statement"`
	Premises    []string    `json:"premises"` // IDs of propositions that this proposition relies upon
	Truth       TruthStatus `json:"truth"`
}

type PropositionSentinelConfig struct {
	MaxRecursionDepth int
	AllowTautology    bool
}

func DefaultPropositionSentinelConfig() PropositionSentinelConfig {
	return PropositionSentinelConfig{
		MaxRecursionDepth: 12,
		AllowTautology:    false,
	}
}

type PropositionSentinelMetrics struct {
	TotalPropositions int    `json:"total_propositions"`
	CyclesIntercepted int    `json:"cycles_intercepted"`
	DepthViolations   int    `json:"depth_violations"`
	ResolutionsCount  int    `json:"resolutions_count"`
	MaxDepthObserved  int    `json:"max_depth_observed"`
	LastViolation     string `json:"last_violation"`
}

type PropositionRecursionSentinel struct {
	mu     sync.RWMutex
	config PropositionSentinelConfig
	nodes  map[string]*PropositionNode

	cyclesIntercepted int
	depthViolations   int
	resolutionsCount  int
	maxDepthObserved  int
	lastViolation     string
}

func NewPropositionRecursionSentinel(cfg PropositionSentinelConfig) *PropositionRecursionSentinel {
	if cfg.MaxRecursionDepth <= 0 {
		cfg.MaxRecursionDepth = 12
	}
	return &PropositionRecursionSentinel{
		config: cfg,
		nodes:  make(map[string]*PropositionNode),
	}
}

func (s *PropositionRecursionSentinel) AddProposition(node PropositionNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if node.ID == "" {
		return errors.New("proposition ID cannot be empty")
	}

	for _, p := range node.Premises {
		if p == node.ID && !s.config.AllowTautology {
			s.cyclesIntercepted++
			s.lastViolation = fmt.Sprintf("self-loop on %s", node.ID)
			return fmt.Errorf("%w: %s refers directly to itself", ErrSelfReferentialProposition, node.ID)
		}
	}

	cleanNode := node
	if cleanNode.Truth == "" {
		cleanNode.Truth = TruthUnknown
	}

	s.nodes[node.ID] = &cleanNode
	return nil
}

// ResolveDependencyChain performs cycle-guarded topological traversal of premises.
// Returns ordered list of evaluation IDs from leaf premises up to target proposition.
func (s *PropositionRecursionSentinel) ResolveDependencyChain(targetID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[targetID]; !exists {
		return nil, fmt.Errorf("%w: %s", ErrPropositionNotFound, targetID)
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	callOrder := make([]string, 0)

	var walk func(currID string, depth int) error
	walk = func(currID string, depth int) error {
		if depth > s.maxDepthObserved {
			s.maxDepthObserved = depth
		}

		if depth > s.config.MaxRecursionDepth {
			s.depthViolations++
			s.lastViolation = fmt.Sprintf("depth %d > limit %d at %s", depth, s.config.MaxRecursionDepth, currID)
			return fmt.Errorf("%w: depth %d on claim %s", ErrPropositionMaxDepthExceeded, depth, currID)
		}

		if recStack[currID] {
			s.cyclesIntercepted++
			s.lastViolation = fmt.Sprintf("cycle detected on re-entering %s", currID)
			return fmt.Errorf("%w: circular premise link on proposition %s", ErrPropositionCycleDetected, currID)
		}

		if visited[currID] {
			return nil
		}

		recStack[currID] = true
		node, ok := s.nodes[currID]
		if ok {
			for _, premiseID := range node.Premises {
				if err := walk(premiseID, depth+1); err != nil {
					return err
				}
			}
		}

		recStack[currID] = false
		visited[currID] = true
		callOrder = append(callOrder, currID)
		return nil
	}

	if err := walk(targetID, 1); err != nil {
		return nil, err
	}

	s.resolutionsCount++
	return callOrder, nil
}

func (s *PropositionRecursionSentinel) GetMetrics() PropositionSentinelMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return PropositionSentinelMetrics{
		TotalPropositions: len(s.nodes),
		CyclesIntercepted: s.cyclesIntercepted,
		DepthViolations:   s.depthViolations,
		ResolutionsCount:  s.resolutionsCount,
		MaxDepthObserved:  s.maxDepthObserved,
		LastViolation:     s.lastViolation,
	}
}
