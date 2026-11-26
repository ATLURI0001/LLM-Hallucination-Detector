package engine

import "fmt"

// AssertClaimCountLimit rejects responses exceeding maxClaims to prevent DoS.
func AssertClaimCountLimit(count, maxClaims int) error {
	if count < 0 || count > maxClaims {
		return fmt.Errorf("sentinel violation: claim count %d exceeds limit %d", count, maxClaims)
	}
	return nil
}
