package engine

import "errors"

// HallucinationCircuitBreaker trips when consecutive hallucination scores exceed threshold.
type HallucinationCircuitBreaker struct {
	maxFail int
	fails   int
	open    bool
}

var ErrHallucinationCircuitOpen = errors.New("hallucination circuit open")

func NewHallucinationCircuitBreaker(maxFail int) *HallucinationCircuitBreaker {
	return &HallucinationCircuitBreaker{maxFail: maxFail}
}

func (h *HallucinationCircuitBreaker) Check(score float64, threshold float64) error {
	if h.open {
		return ErrHallucinationCircuitOpen
	}
	if score > threshold {
		h.fails++
		if h.fails >= h.maxFail {
			h.open = true
		}
		return errors.New("score above threshold")
	}
	h.fails = 0
	return nil
}
