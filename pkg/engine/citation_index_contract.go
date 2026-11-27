package engine

import "fmt"

// ValidateCitationIndex enforces 0 <= idx < sourceCount for citation references.
func ValidateCitationIndex(idx, sourceCount int) error {
	if sourceCount <= 0 {
		return fmt.Errorf("contract violation: sourceCount must be positive")
	}
	if idx < 0 || idx >= sourceCount {
		return fmt.Errorf("contract violation: citation index %d out of range [0, %d)", idx, sourceCount)
	}
	return nil
}
