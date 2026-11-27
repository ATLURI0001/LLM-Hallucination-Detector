package engine

import "fmt"

// HallucinationRateContract enforces a max hallucination rate.
type HallucinationRateContract struct {
	MaxRate float64
}

func NewHallucinationRateContract(maxRate float64) *HallucinationRateContract {
	return &HallucinationRateContract{MaxRate: maxRate}
}

func (c *HallucinationRateContract) Validate(rate float64) error {
	if rate > c.MaxRate {
		return fmt.Errorf("hallucination rate %.2f exceeds max %.2f", rate, c.MaxRate)
	}
	return nil
}
