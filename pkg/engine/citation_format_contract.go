package engine

import (
	"fmt"
	"strings"
)

// CitationFormatContract validates citation bracket format.
type CitationFormatContract struct{}

func NewCitationFormatContract() *CitationFormatContract {
	return &CitationFormatContract{}
}

func (c *CitationFormatContract) Validate(citation string) error {
	if !strings.HasPrefix(citation, "[") || !strings.HasSuffix(citation, "]") {
		return fmt.Errorf("invalid citation format: %s", citation)
	}
	return nil
}
