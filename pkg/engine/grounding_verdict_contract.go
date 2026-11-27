package engine

import "fmt"

// ValidateGroundingVerdict ensures verdict is supported, unsupported, or contradicted.
func ValidateGroundingVerdict(verdict string) error {
	switch verdict {
	case "supported", "unsupported", "contradicted":
		return nil
	default:
		return fmt.Errorf("contract violation: unknown grounding verdict %q", verdict)
	}
}
