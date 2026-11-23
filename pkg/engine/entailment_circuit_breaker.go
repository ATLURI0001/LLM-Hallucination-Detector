package engine

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the current operational status of the entailment breaker.
type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateHalfOpen CircuitState = "HALF_OPEN"
	StateOpen     CircuitState = "OPEN"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is tripped and rejecting primary calls.
	ErrCircuitOpen = errors.New("circuit breaker is open; primary entailment engine unavailable")
	// ErrMaxGraphDepthExceeded is returned when proposition graph traversal exceeds depth safety limit.
	ErrMaxGraphDepthExceeded = errors.New("proposition graph traversal depth exceeded safe bound")
)

// CircuitBreakerConfig defines threshold configurations.
type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	TimeoutDuration  time.Duration
	MaxGraphDepth    int
	NowFunc          func() time.Time
}

// DefaultCircuitBreakerConfig returns resilient production defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		TimeoutDuration:  30 * time.Second,
		MaxGraphDepth:    15,
		NowFunc:          time.Now,
	}
}

// BreakerStats captures live operational counters for telemetry.
type BreakerStats struct {
	State          CircuitState
	TotalRequests  uint64
	TotalFailures  uint64
	TotalSuccesses uint64
	FallbackCount  uint64
	CyclesResolved uint64
}

// EntailmentCircuitBreaker provides fault-tolerance, degraded fallback, and deadlock resolution for entailment graphs.
type EntailmentCircuitBreaker struct {
	mu           sync.RWMutex
	config       CircuitBreakerConfig
	state        CircuitState
	failureCount int
	successCount int
	lastTripped  time.Time

	totalRequests  atomic.Uint64
	totalFailures  atomic.Uint64
	totalSuccesses atomic.Uint64
	fallbackCount  atomic.Uint64
	cyclesResolved atomic.Uint64
}

// NewEntailmentCircuitBreaker initializes a circuit breaker with the given configuration.
func NewEntailmentCircuitBreaker(cfg CircuitBreakerConfig) *EntailmentCircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.TimeoutDuration <= 0 {
		cfg.TimeoutDuration = 30 * time.Second
	}
	if cfg.MaxGraphDepth <= 0 {
		cfg.MaxGraphDepth = 15
	}
	if cfg.NowFunc == nil {
		cfg.NowFunc = time.Now
	}

	return &EntailmentCircuitBreaker{
		config: cfg,
		state:  StateClosed,
	}
}

// Execute wraps an entailment inference call in the circuit breaker state machine.
// If the circuit is open, it returns the conservative fallbackScore with isFallback=true without invoking fn.
func (cb *EntailmentCircuitBreaker) Execute(
	ctx context.Context,
	fn func() (float64, error),
	fallbackScore float64,
) (float64, bool, error) {
	cb.totalRequests.Add(1)

	if !cb.allowExecution() {
		cb.fallbackCount.Add(1)
		return fallbackScore, true, nil
	}

	// Channel execution with context cancellation
	type callResult struct {
		score float64
		err   error
	}

	resCh := make(chan callResult, 1)
	go func() {
		s, err := fn()
		resCh <- callResult{score: s, err: err}
	}()

	select {
	case <-ctx.Done():
		cb.recordFailure()
		cb.fallbackCount.Add(1)
		return fallbackScore, true, ctx.Err()
	case res := <-resCh:
		if res.err != nil {
			cb.recordFailure()
			cb.fallbackCount.Add(1)
			return fallbackScore, true, res.err
		}
		cb.recordSuccess()
		return res.score, false, nil
	}
}

func (cb *EntailmentCircuitBreaker) allowExecution() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.config.NowFunc()
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.Sub(cb.lastTripped) >= cb.config.TimeoutDuration {
			cb.state = StateHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *EntailmentCircuitBreaker) recordSuccess() {
	cb.totalSuccesses.Add(1)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
		}
	} else if cb.state == StateClosed {
		cb.failureCount = 0
	}
}

func (cb *EntailmentCircuitBreaker) recordFailure() {
	cb.totalFailures.Add(1)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	if cb.state == StateHalfOpen || cb.failureCount >= cb.config.FailureThreshold {
		cb.state = StateOpen
		cb.lastTripped = cb.config.NowFunc()
	}
}

// DetectAndResolveCycles analyzes proposition graph adjacency and prunes back-edges that create deadlock loops.
// Returns the topologically sorted evaluation order and a boolean indicating whether cycles were pruned.
func (cb *EntailmentCircuitBreaker) DetectAndResolveCycles(graph map[string][]string, rootNodes []string) ([]string, bool, error) {
	visited := make(map[string]int) // 0: unvisited, 1: visiting (in stack), 2: fully explored
	var order []string
	cyclePruned := false

	var dfs func(node string, depth int) error
	dfs = func(node string, depth int) error {
		if depth > cb.config.MaxGraphDepth {
			return ErrMaxGraphDepthExceeded
		}

		visited[node] = 1 // in current path
		neighbors := graph[node]
		validNeighbors := make([]string, 0, len(neighbors))

		for _, nbr := range neighbors {
			if visited[nbr] == 1 {
				// Cycle detected (back-edge). Prune edge to resolve deadlock!
				cyclePruned = true
				cb.cyclesResolved.Add(1)
				continue // skip adding back-edge
			}

			if visited[nbr] == 0 {
				if err := dfs(nbr, depth+1); err != nil {
					return err
				}
			}
			validNeighbors = append(validNeighbors, nbr)
		}

		graph[node] = validNeighbors
		visited[node] = 2 // finished
		order = append(order, node)
		return nil
	}

	for _, root := range rootNodes {
		if visited[root] == 0 {
			if err := dfs(root, 1); err != nil {
				return nil, cyclePruned, err
			}
		}
	}

	// Reverse order for topological dependency order
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	return order, cyclePruned, nil
}

// GetState returns the current state of the circuit breaker.
func (cb *EntailmentCircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Trip forces the circuit breaker into Open state immediately.
func (cb *EntailmentCircuitBreaker) Trip() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateOpen
	cb.lastTripped = cb.config.NowFunc()
}

// Reset restores the breaker to normal Closed operation.
func (cb *EntailmentCircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
}

// Stats returns a copy of current operational metrics.
func (cb *EntailmentCircuitBreaker) Stats() BreakerStats {
	cb.mu.RLock()
	st := cb.state
	cb.mu.RUnlock()

	return BreakerStats{
		State:          st,
		TotalRequests:  cb.totalRequests.Load(),
		TotalFailures:  cb.totalFailures.Load(),
		TotalSuccesses: cb.totalSuccesses.Load(),
		FallbackCount:  cb.fallbackCount.Load(),
		CyclesResolved: cb.cyclesResolved.Load(),
	}
}
