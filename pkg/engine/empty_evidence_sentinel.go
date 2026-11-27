package engine

import (
	"fmt"
	"strings"
)

// AssertNonEmptyEvidence rejects empty or whitespace-only grounding evidence.
func AssertNonEmptyEvidence(evidence string) error {
	if strings.TrimSpace(evidence) == "" {
		return fmt.Errorf("sentinel violation: grounding evidence must not be empty")
	}
	return nil
}
