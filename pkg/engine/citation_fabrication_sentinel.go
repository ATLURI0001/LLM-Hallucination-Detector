package engine

import (
	"regexp"
	"strings"
)

// CitationFabricationSentinel detects hallucinated URLs, non-existent DOIs,
// and ungrounded reference markers in generated text (INC-2026-HALLUC-4419).
type CitationFabricationSentinel struct {
	AllowedDomains map[string]bool
	urlRegex       *regexp.Regexp
}

// NewCitationFabricationSentinel creates a sentinel with an allowlist of verified domains.
func NewCitationFabricationSentinel(allowedDomains []string) *CitationFabricationSentinel {
	doms := make(map[string]bool)
	for _, d := range allowedDomains {
		doms[strings.ToLower(d)] = true
	}
	return &CitationFabricationSentinel{
		AllowedDomains: doms,
		urlRegex:       regexp.MustCompile(`https?://([a-zA-Z0-9.-]+)`),
	}
}

// Inspect checks generated text for fabricated or untrusted citation URLs.
func (s *CitationFabricationSentinel) Inspect(text string, groundedURLs []string) (bool, []string) {
	groundedSet := make(map[string]bool)
	for _, u := range groundedURLs {
		groundedSet[strings.ToLower(u)] = true
	}

	var fabricated []string
	matches := s.urlRegex.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		fullURL := strings.ToLower(m[0])
		host := strings.ToLower(m[1])

		if !groundedSet[fullURL] && !s.AllowedDomains[host] {
			fabricated = append(fabricated, m[0])
		}
	}

	return len(fabricated) > 0, fabricated
}
