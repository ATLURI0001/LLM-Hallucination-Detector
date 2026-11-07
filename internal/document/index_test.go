package document

import (
	"strings"
	"testing"
)

func TestInvertedIndex(t *testing.T) {
	idx := NewInvertedIndex()

	sourceText := `Google was founded in September 1998 by Larry Page and Sergey Brin while they were Ph.D. students at Stanford University.
The company's initial public offering took place on August 19, 2004.
Sundar Pichai was appointed CEO of Google on October 24, 2015.`

	idx.AddDocument("doc1", sourceText)

	// Search for founders
	res := idx.SearchBM25("Who founded Google at Stanford?", 2)
	if len(res) == 0 {
		t.Fatal("expected search results, got none")
	}

	topChunk := res[0].Chunk.Text
	if !strings.Contains(topChunk, "Larry Page") || !strings.Contains(topChunk, "Sergey Brin") {
		t.Errorf("expected top chunk to mention founders, got: %s", topChunk)
	}

	// Search for IPO
	bestEv := idx.FindBestEvidence("Google held its IPO in August 2004.")
	if bestEv == nil {
		t.Fatal("expected evidence for IPO, got nil")
	}
	if !strings.Contains(bestEv.Snippet, "2004") {
		t.Errorf("expected 2004 in evidence snippet, got: %s", bestEv.Snippet)
	}
}
