package document

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/user/llm-hallucination-detector/internal/models"
)

var (
	wordTokenRegex = regexp.MustCompile(`[\p{L}\p{N}]+`)
)

// Chunk represents a segmented passage of source text with source tracking.
type Chunk struct {
	ID          int
	DocumentID  string
	Text        string
	StartOffset int
	EndOffset   int
	WordCount   int
	TermFreqs   map[string]int
}

// InvertedIndex implements BM25 retrieval across source document chunks.
type InvertedIndex struct {
	chunks      []Chunk
	docCount    int
	avgDocLen   float64
	docFreqs    map[string]int              // Term -> count of chunks containing term
	postings    map[string][]int            // Term -> list of chunk IDs
	k1          float64                     // BM25 term frequency saturation (1.5)
	b           float64                     // BM25 length normalization (0.75)
	stopWords   map[string]bool
}

// NewInvertedIndex initializes an empty search index.
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		docFreqs:  make(map[string]int),
		postings:  make(map[string][]int),
		k1:        1.5,
		b:         0.75,
		stopWords: defaultStopWords(),
	}
}

// AddDocument parses, chunks, and indexes a source document text.
func (idx *InvertedIndex) AddDocument(docID string, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}

	chunks := idx.chunkDocument(docID, text)
	for _, c := range chunks {
		idx.indexChunk(c)
	}

	idx.recomputeStats()
}

// chunkDocument divides document text into sentence-level and sliding window chunks.
func (idx *InvertedIndex) chunkDocument(docID string, text string) []Chunk {
	var chunks []Chunk
	runes := []rune(text)
	n := len(runes)

	type span struct {
		start int
		end   int
		text  string
	}
	var sentences []span

	start := 0
	for i := 0; i < n; i++ {
		r := runes[i]
		isEnd := false
		if r == '.' || r == '!' || r == '?' {
			if i+1 == n || unicode.IsSpace(runes[i+1]) {
				isEnd = true
			}
		} else if r == '\n' {
			if i+1 < n && runes[i+1] == '\n' {
				isEnd = true
			}
		}

		if isEnd && i >= start {
			s := strings.TrimSpace(string(runes[start : i+1]))
			if len(s) > 5 {
				sentences = append(sentences, span{
					start: start,
					end:   i + 1,
					text:  s,
				})
			}
			start = i + 1
		}
	}
	if start < n {
		s := strings.TrimSpace(string(runes[start:n]))
		if len(s) > 5 {
			sentences = append(sentences, span{
				start: start,
				end:   n,
				text:  s,
			})
		}
	}

	chunkID := len(idx.chunks) + 1

	// Single sentence chunks
	for _, sent := range sentences {
		tokens := idx.Tokenize(sent.text)
		tf := make(map[string]int)
		for _, t := range tokens {
			tf[t]++
		}
		chunks = append(chunks, Chunk{
			ID:          chunkID,
			DocumentID:  docID,
			Text:        sent.text,
			StartOffset: sent.start,
			EndOffset:   sent.end,
			WordCount:   len(tokens),
			TermFreqs:   tf,
		})
		chunkID++
	}

	// 2-sentence sliding window chunks
	for i := 0; i < len(sentences)-1; i++ {
		s1 := sentences[i]
		s2 := sentences[i+1]
		combinedText := strings.TrimSpace(string(runes[s1.start:s2.end]))
		tokens := idx.Tokenize(combinedText)
		tf := make(map[string]int)
		for _, t := range tokens {
			tf[t]++
		}
		chunks = append(chunks, Chunk{
			ID:          chunkID,
			DocumentID:  docID,
			Text:        combinedText,
			StartOffset: s1.start,
			EndOffset:   s2.end,
			WordCount:   len(tokens),
			TermFreqs:   tf,
		})
		chunkID++
	}

	return chunks
}

func (idx *InvertedIndex) indexChunk(c Chunk) {
	idx.chunks = append(idx.chunks, c)
	seen := make(map[string]bool)
	for term := range c.TermFreqs {
		if !seen[term] {
			seen[term] = true
			idx.docFreqs[term]++
			idx.postings[term] = append(idx.postings[term], c.ID)
		}
	}
}

func (idx *InvertedIndex) recomputeStats() {
	idx.docCount = len(idx.chunks)
	if idx.docCount == 0 {
		idx.avgDocLen = 0
		return
	}
	totalWords := 0
	for _, c := range idx.chunks {
		totalWords += c.WordCount
	}
	idx.avgDocLen = float64(totalWords) / float64(idx.docCount)
}

// ScoredChunk is a candidate evidence chunk with BM25 score.
type ScoredChunk struct {
	Chunk Chunk
	Score float64
}

// SearchBM25 returns ranked candidate evidence chunks for a given query text.
func (idx *InvertedIndex) SearchBM25(query string, topK int) []ScoredChunk {
	if idx.docCount == 0 || topK <= 0 {
		return nil
	}

	queryTokens := idx.Tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	scores := make(map[int]float64)

	for _, token := range queryTokens {
		df := idx.docFreqs[token]
		if df == 0 {
			continue
		}

		idf := math.Log(1.0 + (float64(idx.docCount)-float64(df)+0.5)/(float64(df)+0.5))
		if idf < 0.1 {
			idf = 0.1
		}

		for _, chunkID := range idx.postings[token] {
			c := idx.chunks[chunkID-1]
			tf := float64(c.TermFreqs[token])
			docLen := float64(c.WordCount)

			numerator := tf * (idx.k1 + 1.0)
			denominator := tf + idx.k1*(1.0-idx.b+idx.b*(docLen/idx.avgDocLen))
			bm25Term := idf * (numerator / denominator)

			scores[chunkID] += bm25Term
		}
	}

	var ranked []ScoredChunk
	for chunkID, score := range scores {
		ranked = append(ranked, ScoredChunk{
			Chunk: idx.chunks[chunkID-1],
			Score: score,
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	if len(ranked) > topK {
		ranked = ranked[:topK]
	}

	return ranked
}

// FindBestEvidence retrieves the most relevant passage for a claim.
func (idx *InvertedIndex) FindBestEvidence(claimText string) *models.Evidence {
	results := idx.SearchBM25(claimText, 3)
	if len(results) == 0 {
		return nil
	}

	top := results[0]
	if top.Score <= 0 {
		return nil
	}

	normScore := top.Score / (top.Score + 1.5)

	exact := strings.Contains(strings.ToLower(top.Chunk.Text), strings.ToLower(strings.TrimSpace(claimText)))

	return &models.Evidence{
		DocumentID:      top.Chunk.DocumentID,
		Snippet:         top.Chunk.Text,
		SourceStart:     top.Chunk.StartOffset,
		SourceEnd:       top.Chunk.EndOffset,
		SimilarityScore: normScore,
		ExactMatch:      exact,
	}
}

// Tokenize converts text into normalized lowercase tokens without punctuation.
func (idx *InvertedIndex) Tokenize(text string) []string {
	words := wordTokenRegex.FindAllString(strings.ToLower(text), -1)
	var tokens []string
	for _, w := range words {
		if !idx.stopWords[w] && len(w) > 1 {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func defaultStopWords() map[string]bool {
	words := []string{
		"a", "about", "above", "after", "again", "against", "all", "am", "an", "and",
		"any", "are", "aren't", "as", "at", "be", "because", "been", "before", "being",
		"below", "between", "both", "but", "by", "can't", "cannot", "could", "couldn't",
		"did", "didn't", "do", "does", "doesn't", "doing", "don't", "down", "during",
		"each", "few", "for", "from", "further", "had", "hadn't", "has", "hasn't",
		"have", "haven't", "having", "he", "he'd", "he'll", "he's", "her", "here",
		"here's", "hers", "herself", "him", "himself", "his", "how", "how's", "i",
		"i'd", "i'll", "i'm", "i've", "if", "in", "into", "is", "isn't", "it",
		"it's", "its", "itself", "let's", "me", "more", "most", "mustn't", "my",
		"myself", "no", "nor", "not", "of", "off", "on", "once", "only", "or",
		"other", "ought", "our", "ours", "ourselves", "out", "over", "own", "same",
		"shan't", "she", "she'd", "she'll", "she's", "should", "shouldn't", "so",
		"some", "such", "than", "that", "that's", "the", "their", "theirs", "them",
		"themselves", "then", "there", "there's", "these", "they", "they'd", "they'll",
		"they're", "they've", "this", "those", "through", "to", "too", "under", "until",
		"up", "very", "was", "wasn't", "we", "we'd", "we'll", "we're", "we've",
		"were", "weren't", "what", "what's", "when", "when's", "where", "where's",
		"which", "while", "who", "who's", "whom", "why", "why's", "with", "won't",
		"would", "wouldn't", "you", "you'd", "you'll", "you're", "you've", "your",
		"yours", "yourself", "yourselves",
	}
	m := make(map[string]bool)
	for _, w := range words {
		m[w] = true
	}
	return m
}
