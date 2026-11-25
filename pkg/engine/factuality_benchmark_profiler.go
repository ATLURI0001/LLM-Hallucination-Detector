package engine

import (
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// FactualityResult represents verification outcome between a claim and reference context.
type FactualityResult struct {
	ClaimID        string
	ReferenceID    string
	EntailmentProb float64
	IsSupported    bool
	LatencyNs      int64
}

// FactualityBenchmarkReport contains detailed statistical latency and throughput metrics.
type FactualityBenchmarkReport struct {
	TotalEvaluations    int
	Concurrency         int
	TotalDuration       time.Duration
	ThroughputOpsPerSec float64
	MeanLatencyNs       int64
	P50LatencyNs        int64
	P90LatencyNs        int64
	P99LatencyNs        int64
	MinLatencyNs        int64
	MaxLatencyNs        int64
	AllocatedBytes      uint64
	AllocsCount         uint64
	SupportedRatio      float64
}

// FactualityBenchmarkProfiler executes concurrent verification benchmarks across claim matrices.
type FactualityBenchmarkProfiler struct {
	mu sync.RWMutex
}

// NewFactualityBenchmarkProfiler constructs a profiler instance.
func NewFactualityBenchmarkProfiler() *FactualityBenchmarkProfiler {
	return &FactualityBenchmarkProfiler{}
}

// VerifyClaimPair evaluates lexical token alignment and overlap as a fast factuality surrogate.
func (p *FactualityBenchmarkProfiler) VerifyClaimPair(claim, reference string) (float64, bool) {
	claimWords := extractFactualityWords(claim)
	refWords := extractFactualityWords(reference)

	if len(claimWords) == 0 || len(refWords) == 0 {
		return 0.0, false
	}

	refSet := make(map[string]struct{}, len(refWords))
	for _, w := range refWords {
		refSet[w] = struct{}{}
	}

	matchCount := 0
	for _, w := range claimWords {
		if _, ok := refSet[w]; ok {
			matchCount++
		}
	}

	ratio := float64(matchCount) / float64(len(claimWords))
	isSupported := ratio >= 0.5
	return math.Round(ratio*1000) / 1000, isSupported
}

func extractFactualityWords(s string) []string {
	fields := strings.Fields(strings.ToLower(s))
	clean := make([]string, 0, len(fields))
	for _, f := range fields {
		trimmed := strings.Trim(f, ".,!?:;\"'()[]{}")
		if len(trimmed) > 1 {
			clean = append(clean, trimmed)
		}
	}
	return clean
}

// RunBenchmark profiles claim verification matrix performance under load.
func (p *FactualityBenchmarkProfiler) RunBenchmark(claims, references []string, concurrency int, repetitions int) FactualityBenchmarkReport {
	if concurrency <= 0 {
		concurrency = 1
	}
	if repetitions <= 0 {
		repetitions = 1
	}
	if len(claims) == 0 || len(references) == 0 {
		return FactualityBenchmarkReport{}
	}

	// Warmup run
	for i := 0; i < 50; i++ {
		p.VerifyClaimPair(claims[i%len(claims)], references[i%len(references)])
	}

	type task struct {
		claim string
		ref   string
	}

	totalPairs := len(claims) * len(references) * repetitions
	tasks := make(chan task, totalPairs)
	for r := 0; r < repetitions; r++ {
		for _, c := range claims {
			for _, ref := range references {
				tasks <- task{claim: c, ref: ref}
			}
		}
	}
	close(tasks)

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	wallStart := time.Now()

	latencies := make([]int64, totalPairs)
	var latIndex atomicInt
	var supportedCount atomicInt

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for t := range tasks {
				opStart := time.Now()
				prob, isSup := p.VerifyClaimPair(t.claim, t.ref)
				elapsed := time.Since(opStart).Nanoseconds()

				idx := latIndex.Add(1) - 1
				if int(idx) < len(latencies) {
					latencies[idx] = elapsed
				}
				if isSup && prob > 0.5 {
					supportedCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()
	wallDuration := time.Since(wallStart)
	if wallDuration <= 0 {
		wallDuration = time.Microsecond
	}
	elapsedNs := wallDuration.Nanoseconds()

	runtime.ReadMemStats(&memAfter)

	sorted := make([]int64, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	getP := func(pct float64) int64 {
		if len(sorted) == 0 {
			return 0
		}
		idx := int(math.Ceil(pct*float64(len(sorted)))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return sorted[idx]
	}

	var sumLat int64
	for _, l := range sorted {
		sumLat += l
	}
	meanLat := int64(0)
	if len(sorted) > 0 {
		meanLat = sumLat / int64(len(sorted))
	}

	opsPerSec := float64(totalPairs) / (float64(elapsedNs) / 1e9)

	supportedRatio := 0.0
	if totalPairs > 0 {
		supportedRatio = float64(supportedCount.Load()) / float64(totalPairs)
	}

	var allocBytes uint64
	if memAfter.TotalAlloc > memBefore.TotalAlloc {
		allocBytes = memAfter.TotalAlloc - memBefore.TotalAlloc
	}

	var allocCount uint64
	if memAfter.Mallocs > memBefore.Mallocs {
		allocCount = memAfter.Mallocs - memBefore.Mallocs
	}

	minLat := int64(0)
	maxLat := int64(0)
	if len(sorted) > 0 {
		minLat = sorted[0]
		maxLat = sorted[len(sorted)-1]
	}

	return FactualityBenchmarkReport{
		TotalEvaluations:    totalPairs,
		Concurrency:         concurrency,
		TotalDuration:       wallDuration,
		ThroughputOpsPerSec: math.Round(opsPerSec*100) / 100,
		MeanLatencyNs:       meanLat,
		P50LatencyNs:        getP(0.50),
		P90LatencyNs:        getP(0.90),
		P99LatencyNs:        getP(0.99),
		MinLatencyNs:        minLat,
		MaxLatencyNs:        maxLat,
		AllocatedBytes:      allocBytes,
		AllocsCount:         allocCount,
		SupportedRatio:      math.Round(supportedRatio*1000) / 1000,
	}
}

type atomicInt struct {
	val int64
	mu  sync.Mutex
}

func (a *atomicInt) Add(n int64) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.val += n
	return a.val
}

func (a *atomicInt) Load() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.val
}
