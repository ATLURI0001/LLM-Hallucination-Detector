package claim

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	manifestIDRegex = regexp.MustCompile(`^cite-man-[a-zA-Z0-9_-]{4,64}$`)
	documentIDRegex = regexp.MustCompile(`^doc-[a-zA-Z0-9_-]{3,64}$`)
	citationIDRegex = regexp.MustCompile(`^cite-[a-zA-Z0-9_-]{2,32}$`)
	chunkIDRegex    = regexp.MustCompile(`^chunk-[a-zA-Z0-9_-]{2,32}$`)
	invisibleChars  = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F\x{200B}-\x{200D}\x{FEFF}]`)
)

// CitationEntry represents an individual verified reference source citation.
type CitationEntry struct {
	CitationID      string  `json:"citation_id"`
	SourceChunkID   string  `json:"source_chunk_id"`
	ExactText       string  `json:"exact_text"`
	CharOffsetStart int     `json:"char_offset_start"`
	CharOffsetEnd   int     `json:"char_offset_end"`
	ConfidenceScore float64 `json:"confidence_score"`
	ContextSpan     string  `json:"context_span,omitempty"`
}

// CitationManifest represents an ingress batch of document reference citations.
type CitationManifest struct {
	ManifestID string          `json:"manifest_id"`
	DocumentID string          `json:"document_id"`
	Citations  []CitationEntry `json:"citations"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ValidatedCitationManifest represents a sanitized, verified, and checksummed citation manifest.
type ValidatedCitationManifest struct {
	ManifestID       string          `json:"manifest_id"`
	DocumentID       string          `json:"document_id"`
	Citations        []CitationEntry `json:"citations"`
	Checksum         string          `json:"checksum"`
	ValidatedAt      time.Time       `json:"validated_at"`
	TotalCitations   int             `json:"total_citations"`
	MeanConfidence   float64         `json:"mean_confidence"`
}

// ContractValidationError wraps boundary violations with explicit codes and field names.
type ContractValidationError struct {
	Code    string
	Field   string
	Message string
}

func (e *ContractValidationError) Error() string {
	return fmt.Sprintf("[%s] validation error on field '%s': %s", e.Code, e.Field, e.Message)
}

// SanitizeCitationText removes non-printable control characters and zero-width spaces.
func SanitizeCitationText(input string) string {
	return strings.TrimSpace(invisibleChars.ReplaceAllString(input, ""))
}

// ValidateCitationManifest enforces structural invariants, monotonic offset ranges, and checksum integrity.
func ValidateCitationManifest(manifest *CitationManifest) (*ValidatedCitationManifest, error) {
	if manifest == nil {
		return nil, &ContractValidationError{
			Code:    "ERR_NIL_MANIFEST",
			Field:   "manifest",
			Message: "manifest pointer cannot be nil",
		}
	}

	// 1. Manifest ID
	manifestID := strings.TrimSpace(manifest.ManifestID)
	if !manifestIDRegex.MatchString(manifestID) {
		return nil, &ContractValidationError{
			Code:    "ERR_INVALID_MANIFEST_ID",
			Field:   "manifest_id",
			Message: fmt.Sprintf("manifest_id must match 'cite-man-<id>' [4-64 chars], got '%s'", manifestID),
		}
	}

	// 2. Document ID
	documentID := strings.TrimSpace(manifest.DocumentID)
	if !documentIDRegex.MatchString(documentID) {
		return nil, &ContractValidationError{
			Code:    "ERR_INVALID_DOC_ID",
			Field:   "document_id",
			Message: fmt.Sprintf("document_id must match 'doc-<id>' [3-64 chars], got '%s'", documentID),
		}
	}

	// 3. Citation slice bounds
	nCitations := len(manifest.Citations)
	if nCitations == 0 {
		return nil, &ContractValidationError{
			Code:    "ERR_EMPTY_CITATIONS",
			Field:   "citations",
			Message: "citation manifest must contain at least 1 citation entry",
		}
	}
	if nCitations > 200 {
		return nil, &ContractValidationError{
			Code:    "ERR_CITATIONS_OVERFLOW",
			Field:   "citations",
			Message: fmt.Sprintf("citation manifest exceeds maximum capacity of 200 (got %d)", nCitations),
		}
	}

	seenCitationIDs := make(map[string]struct{}, nCitations)
	sanitizedCitations := make([]CitationEntry, 0, nCitations)
	var sumConfidence float64

	// 4. Validate each citation entry
	for i, c := range manifest.Citations {
		fieldPrefix := fmt.Sprintf("citations[%d]", i)

		// Citation ID
		cid := strings.TrimSpace(c.CitationID)
		if !citationIDRegex.MatchString(cid) {
			return nil, &ContractValidationError{
				Code:    "ERR_INVALID_CITATION_ID",
				Field:   fieldPrefix + ".citation_id",
				Message: fmt.Sprintf("citation_id must match 'cite-<id>' pattern, got '%s'", cid),
			}
		}
		if _, exists := seenCitationIDs[cid]; exists {
			return nil, &ContractValidationError{
				Code:    "ERR_DUPLICATE_CITATION_ID",
				Field:   fieldPrefix + ".citation_id",
				Message: fmt.Sprintf("duplicate citation ID '%s' found in manifest", cid),
			}
		}
		seenCitationIDs[cid] = struct{}{}

		// Source Chunk ID
		chunkID := strings.TrimSpace(c.SourceChunkID)
		if !chunkIDRegex.MatchString(chunkID) {
			return nil, &ContractValidationError{
				Code:    "ERR_INVALID_CHUNK_ID",
				Field:   fieldPrefix + ".source_chunk_id",
				Message: fmt.Sprintf("source_chunk_id must match 'chunk-<id>' pattern, got '%s'", chunkID),
			}
		}

		// ExactText
		cleanText := SanitizeCitationText(c.ExactText)
		if len(cleanText) < 5 {
			return nil, &ContractValidationError{
				Code:    "ERR_EXACT_TEXT_TOO_SHORT",
				Field:   fieldPrefix + ".exact_text",
				Message: "exact_text must contain at least 5 printable characters",
			}
		}
		if len(cleanText) > 2000 {
			return nil, &ContractValidationError{
				Code:    "ERR_EXACT_TEXT_TOO_LONG",
				Field:   fieldPrefix + ".exact_text",
				Message: fmt.Sprintf("exact_text exceeds 2,000 characters limit (got %d)", len(cleanText)),
			}
		}

		// Char Offsets
		if c.CharOffsetStart < 0 {
			return nil, &ContractValidationError{
				Code:    "ERR_NEGATIVE_OFFSET",
				Field:   fieldPrefix + ".char_offset_start",
				Message: "char_offset_start cannot be negative",
			}
		}
		if c.CharOffsetEnd <= c.CharOffsetStart {
			return nil, &ContractValidationError{
				Code:    "ERR_INVALID_OFFSET_RANGE",
				Field:   fieldPrefix + ".char_offset_end",
				Message: fmt.Sprintf("char_offset_end (%d) must be strictly greater than char_offset_start (%d)", c.CharOffsetEnd, c.CharOffsetStart),
			}
		}

		spanLen := c.CharOffsetEnd - c.CharOffsetStart
		if spanLen < len(cleanText) {
			return nil, &ContractValidationError{
				Code:    "ERR_OFFSET_SPAN_MISMATCH",
				Field:   fieldPrefix + ".char_offset_end",
				Message: fmt.Sprintf("offset span length (%d) is smaller than exact_text length (%d)", spanLen, len(cleanText)),
			}
		}

		// Confidence Score
		if c.ConfidenceScore < 0.0 || c.ConfidenceScore > 1.0 {
			return nil, &ContractValidationError{
				Code:    "ERR_INVALID_CONFIDENCE",
				Field:   fieldPrefix + ".confidence_score",
				Message: fmt.Sprintf("confidence_score must be within [0.0, 1.0], got %f", c.ConfidenceScore),
			}
		}
		sumConfidence += c.ConfidenceScore

		cleanContext := SanitizeCitationText(c.ContextSpan)
		if len(cleanContext) > 10000 {
			cleanContext = cleanContext[:10000]
		}

		sanitizedCitations = append(sanitizedCitations, CitationEntry{
			CitationID:      cid,
			SourceChunkID:   chunkID,
			ExactText:       cleanText,
			CharOffsetStart: c.CharOffsetStart,
			CharOffsetEnd:   c.CharOffsetEnd,
			ConfidenceScore: c.ConfidenceScore,
			ContextSpan:     cleanContext,
		})
	}

	meanConfidence := sumConfidence / float64(nCitations)

	// Canonical JSON serialization for checksum computation
	canonicalBytes, err := json.Marshal(struct {
		ManifestID string          `json:"manifest_id"`
		DocumentID string          `json:"document_id"`
		Citations  []CitationEntry `json:"citations"`
	}{
		ManifestID: manifestID,
		DocumentID: documentID,
		Citations:  sanitizedCitations,
	})
	if err != nil {
		return nil, errors.New("failed to canonicalize manifest for checksum calculation")
	}

	hash := sha256.Sum256(canonicalBytes)
	checksum := hex.EncodeToString(hash[:])

	return &ValidatedCitationManifest{
		ManifestID:     manifestID,
		DocumentID:     documentID,
		Citations:      sanitizedCitations,
		Checksum:       checksum,
		ValidatedAt:    time.Now().UTC(),
		TotalCitations: nCitations,
		MeanConfidence: meanConfidence,
	}, nil
}
