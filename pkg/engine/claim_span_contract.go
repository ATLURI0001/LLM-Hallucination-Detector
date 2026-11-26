package engine

import "fmt"

// ClaimSpan represents a character span [Start, End) in a response.
type ClaimSpan struct {
	Text  string
	Start int
	End   int
}

// ValidateClaimSpan enforces valid non-empty claim span bounds against sourceLen.
func ValidateClaimSpan(span ClaimSpan, sourceLen int) error {
	if span.Text == "" {
		return fmt.Errorf("contract violation: claim text cannot be empty")
	}
	if span.Start < 0 || span.End <= span.Start || span.End > sourceLen {
		return fmt.Errorf("contract violation: invalid span [%d, %d) for sourceLen %d", span.Start, span.End, sourceLen)
	}
	return nil
}
