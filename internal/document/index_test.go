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

func TestSearchBM25ZeroAvgDocLenAndDeterministicTieBreak(t *testing.T) {
	// Corpus where chunks contain identical token frequency -> tied scores must stay deterministic
	idx := NewInvertedIndex()
	idx.AddDocument("doc1", "Alpha service responds cleanly. Alpha service recovers quickly.")
	first := idx.SearchBM25("Alpha", 5)
	if len(first) < 2 {
		t.Fatalf("expected at least 2 ranked chunks, got %d", len(first))
	}
	for i := 0; i < 20; i++ {
		next := idx.SearchBM25("Alpha", 5)
		if next[0].Chunk.ID != first[0].Chunk.ID || next[1].Chunk.ID != first[1].Chunk.ID {
			t.Fatalf("expected deterministic tie-breaking, iteration %d diverged", i)
		}
	}

	// Zero-length/stopword chunks should never produce NaN scores when searched
	stopIdx := NewInvertedIndex()
	stopIdx.chunks = []Chunk{{ID: 1, DocumentID: "d", Text: "alpha beta", WordCount: 0, TermFreqs: map[string]int{"alpha": 1}}}
	stopIdx.docCount = 1
	stopIdx.avgDocLen = 0
	stopIdx.docFreqs["alpha"] = 1
	stopIdx.postings["alpha"] = []int{1}
	res := stopIdx.SearchBM25("alpha", 1)
	if len(res) != 1 || res[0].Score <= 0 {
		t.Fatalf("expected positive non-NaN score on zero avgDocLen chunk, got %+v", res)
	}
}
