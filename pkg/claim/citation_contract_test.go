package claim

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateCitationManifest_Success(t *testing.T) {
	manifest := &CitationManifest{
		ManifestID: "cite-man-alpha-1234",
		DocumentID: "doc-quarterly-report-2026",
		Citations: []CitationEntry{
			{
				CitationID:      "cite-001",
				SourceChunkID:   "chunk-sec-1",
				ExactText:       "Gross revenue grew by 14.2% year-over-year.",
				CharOffsetStart: 100,
				CharOffsetEnd:   160,
				ConfidenceScore: 0.98,
				ContextSpan:     "In the fourth quarter, gross revenue grew by 14.2% year-over-year driven by cloud expansion.",
			},
			{
				CitationID:      "cite-002",
				SourceChunkID:   "chunk-sec-2",
				ExactText:       "Operating margin expanded to 28.5%.",
				CharOffsetStart: 250,
				CharOffsetEnd:   300,
				ConfidenceScore: 0.94,
			},
		},
	}

	validated, err := ValidateCitationManifest(manifest)
	if err != nil {
		t.Fatalf("unexpected error validating compliant manifest: %v", err)
	}

	if validated.ManifestID != "cite-man-alpha-1234" {
		t.Errorf("expected manifest ID cite-man-alpha-1234, got %s", validated.ManifestID)
	}
	if validated.TotalCitations != 2 {
		t.Errorf("expected 2 citations, got %d", validated.TotalCitations)
	}
	if len(validated.Checksum) != 64 {
		t.Errorf("expected 64-char SHA-256 checksum, got %s", validated.Checksum)
	}
	if validated.MeanConfidence < 0.95 || validated.MeanConfidence > 0.97 {
		t.Errorf("expected mean confidence around 0.96, got %f", validated.MeanConfidence)
	}
}

func TestValidateCitationManifest_NilAndIdentifiers(t *testing.T) {
	_, err := ValidateCitationManifest(nil)
	if err == nil {
		t.Fatal("expected error on nil manifest")
	}

	// Invalid manifest ID
	mBadID := &CitationManifest{
		ManifestID: "invalid-id",
		DocumentID: "doc-valid-01",
		Citations:  []CitationEntry{{CitationID: "cite-01", SourceChunkID: "chunk-01", ExactText: "valid text", CharOffsetStart: 0, CharOffsetEnd: 20, ConfidenceScore: 0.9}},
	}
	_, err = ValidateCitationManifest(mBadID)
	var valErr *ContractValidationError
	if !errors.As(err, &valErr) || valErr.Code != "ERR_INVALID_MANIFEST_ID" {
		t.Errorf("expected ERR_INVALID_MANIFEST_ID, got %v", err)
	}

	// Invalid doc ID
	mBadDoc := &CitationManifest{
		ManifestID: "cite-man-1234",
		DocumentID: "bad_doc",
		Citations:  []CitationEntry{{CitationID: "cite-01", SourceChunkID: "chunk-01", ExactText: "valid text", CharOffsetStart: 0, CharOffsetEnd: 20, ConfidenceScore: 0.9}},
	}
	_, err = ValidateCitationManifest(mBadDoc)
	if !errors.As(err, &valErr) || valErr.Code != "ERR_INVALID_DOC_ID" {
		t.Errorf("expected ERR_INVALID_DOC_ID, got %v", err)
	}
}

func TestValidateCitationManifest_CitationListBounds(t *testing.T) {
	var valErr *ContractValidationError

	// Empty list
	mEmpty := &CitationManifest{
		ManifestID: "cite-man-1234",
		DocumentID: "doc-valid-01",
		Citations:  []CitationEntry{},
	}
	_, err := ValidateCitationManifest(mEmpty)
	if !errors.As(err, &valErr) || valErr.Code != "ERR_EMPTY_CITATIONS" {
		t.Errorf("expected ERR_EMPTY_CITATIONS, got %v", err)
	}

	// Overflow > 200
	mOverflow := &CitationManifest{
		ManifestID: "cite-man-1234",
		DocumentID: "doc-valid-01",
		Citations:  make([]CitationEntry, 205),
	}
	_, err = ValidateCitationManifest(mOverflow)
	if !errors.As(err, &valErr) || valErr.Code != "ERR_CITATIONS_OVERFLOW" {
		t.Errorf("expected ERR_CITATIONS_OVERFLOW, got %v", err)
	}
}

func TestValidateCitationManifest_OffsetsAndSpans(t *testing.T) {
	var valErr *ContractValidationError

	// Negative offset start
	mNeg := &CitationManifest{
		ManifestID: "cite-man-1234",
		DocumentID: "doc-valid-01",
		Citations: []CitationEntry{
			{CitationID: "cite-01", SourceChunkID: "chunk-01", ExactText: "valid text", CharOffsetStart: -5, CharOffsetEnd: 20, ConfidenceScore: 0.9},
		},
	}
	_, err := ValidateCitationManifest(mNeg)
	if !errors.As(err, &valErr) || valErr.Code != "ERR_NEGATIVE_OFFSET" {
		t.Errorf("expected ERR_NEGATIVE_OFFSET, got %v", err)
	}

	// OffsetEnd <= OffsetStart
	mInverted := &CitationManifest{
		ManifestID: "cite-man-1234",
		DocumentID: "doc-valid-01",
		Citations: []CitationEntry{
			{CitationID: "cite-01", SourceChunkID: "chunk-01", ExactText: "valid text", CharOffsetStart: 25, CharOffsetEnd: 20, ConfidenceScore: 0.9},
		},
	}
	_, err = ValidateCitationManifest(mInverted)
	if !errors.As(err, &valErr) || valErr.Code != "ERR_INVALID_OFFSET_RANGE" {
		t.Errorf("expected ERR_INVALID_OFFSET_RANGE, got %v", err)
	}

	// Span shorter than text
	mShortSpan := &CitationManifest{
		ManifestID: "cite-man-1234",
		DocumentID: "doc-valid-01",
		Citations: []CitationEntry{
			{CitationID: "cite-01", SourceChunkID: "chunk-01", ExactText: "this is long verbatim text", CharOffsetStart: 10, CharOffsetEnd: 15, ConfidenceScore: 0.9},
		},
	}
	_, err = ValidateCitationManifest(mShortSpan)
	if !errors.As(err, &valErr) || valErr.Code != "ERR_OFFSET_SPAN_MISMATCH" {
		t.Errorf("expected ERR_OFFSET_SPAN_MISMATCH, got %v", err)
	}
}

func TestValidateCitationManifest_DuplicateCitationID(t *testing.T) {
	var valErr *ContractValidationError

	mDup := &CitationManifest{
		ManifestID: "cite-man-1234",
		DocumentID: "doc-valid-01",
		Citations: []CitationEntry{
			{CitationID: "cite-01", SourceChunkID: "chunk-01", ExactText: "first excerpt", CharOffsetStart: 0, CharOffsetEnd: 30, ConfidenceScore: 0.9},
			{CitationID: "cite-01", SourceChunkID: "chunk-02", ExactText: "second excerpt", CharOffsetStart: 40, CharOffsetEnd: 70, ConfidenceScore: 0.8},
		},
	}
	_, err := ValidateCitationManifest(mDup)
	if !errors.As(err, &valErr) || valErr.Code != "ERR_DUPLICATE_CITATION_ID" {
		t.Errorf("expected ERR_DUPLICATE_CITATION_ID, got %v", err)
	}
}

func TestValidateCitationManifest_SanitizationAndChecksum(t *testing.T) {
	// Invisible unicode in text
	dirtyText := "Clean\u200B\uFEFF Text With Excerpt"
	manifest := &CitationManifest{
		ManifestID: "cite-man-1234",
		DocumentID: "doc-valid-01",
		Citations: []CitationEntry{
			{CitationID: "cite-01", SourceChunkID: "chunk-01", ExactText: dirtyText, CharOffsetStart: 0, CharOffsetEnd: 50, ConfidenceScore: 0.85},
		},
	}

	validated, err := ValidateCitationManifest(manifest)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if strings.Contains(validated.Citations[0].ExactText, "\u200B") {
		t.Errorf("expected zero-width space to be stripped, got %s", validated.Citations[0].ExactText)
	}
	if validated.Citations[0].ExactText != "Clean Text With Excerpt" {
		t.Errorf("expected sanitized text 'Clean Text With Excerpt', got '%s'", validated.Citations[0].ExactText)
	}
}
