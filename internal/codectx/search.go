package codectx

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// BM25 parameters. k1 controls term-frequency saturation, b controls
// length normalisation; these are the standard defaults.
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

type bm25Stats struct {
	docFreq map[string]int
	avgLen  float64
	total   int
}

// tokenize splits text into lowercase terms, additionally emitting the
// pieces of camelCase and snake_case identifiers so a query for "auth"
// matches "useAuthToken".
func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	out := make([]string, 0, len(fields)*2)
	for _, f := range fields {
		if len(f) < 2 || len(f) > 40 {
			continue
		}
		out = append(out, f)
	}

	// Split identifiers on case transitions from the raw (non-lowered) text.
	for _, f := range strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		for _, part := range splitCamel(f) {
			p := strings.ToLower(part)
			if len(p) >= 2 && len(p) <= 40 && p != strings.ToLower(f) {
				out = append(out, p)
			}
		}
	}
	return out
}

func splitCamel(s string) []string {
	var parts []string
	start := 0
	for i := 1; i < len(s); i++ {
		if unicode.IsUpper(rune(s[i])) && !unicode.IsUpper(rune(s[i-1])) {
			parts = append(parts, s[start:i])
			start = i
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	if len(parts) < 2 {
		return nil
	}
	return parts
}

// prepare computes per-chunk token frequencies and corpus statistics.
// Called after loading or building an index.
func (ix *Index) prepare() {
	df := map[string]int{}
	totalLen := 0

	for i := range ix.Chunks {
		c := &ix.Chunks[i]
		// Include the path so filename matches score highly.
		c.tokens = tokenize(c.Path + " " + c.Text)
		c.tokenFreq = make(map[string]int, len(c.tokens))
		for _, t := range c.tokens {
			c.tokenFreq[t]++
		}
		for t := range c.tokenFreq {
			df[t]++
		}
		totalLen += len(c.tokens)
	}

	avg := 0.0
	if len(ix.Chunks) > 0 {
		avg = float64(totalLen) / float64(len(ix.Chunks))
	}
	ix.bm25 = &bm25Stats{docFreq: df, avgLen: avg, total: len(ix.Chunks)}
}

// Hit is one retrieved chunk with its relevance score.
type Hit struct {
	Chunk Chunk   `json:"chunk"`
	Score float64 `json:"score"`
}

// Search returns the top-k chunks for a natural-language query.
func (ix *Index) Search(query string, k int) []Hit {
	if ix.bm25 == nil {
		ix.prepare()
	}
	if ix.bm25.total == 0 || k <= 0 {
		return nil
	}

	terms := dedupe(tokenize(query))
	if len(terms) == 0 {
		return nil
	}

	N := float64(ix.bm25.total)
	scored := make([]Hit, 0, len(ix.Chunks))

	for i := range ix.Chunks {
		c := &ix.Chunks[i]
		docLen := float64(len(c.tokens))
		var score float64
		for _, t := range terms {
			tf := float64(c.tokenFreq[t])
			if tf == 0 {
				continue
			}
			n := float64(ix.bm25.docFreq[t])
			idf := math.Log(1 + (N-n+0.5)/(n+0.5))
			norm := tf * (bm25K1 + 1) /
				(tf + bm25K1*(1-bm25B+bm25B*docLen/ix.bm25.avgLen))
			score += idf * norm
		}
		if score > 0 {
			scored = append(scored, Hit{Chunk: *c, Score: score})
		}
	}

	sort.Slice(scored, func(a, b int) bool { return scored[a].Score > scored[b].Score })
	if len(scored) > k {
		scored = scored[:k]
	}
	return dropDuplicateFiles(scored, k)
}

// dropDuplicateFiles caps how many chunks any single file contributes, so
// one large file cannot crowd out the rest of the codebase.
func dropDuplicateFiles(hits []Hit, k int) []Hit {
	const perFile = 3
	count := map[string]int{}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		if count[h.Chunk.Path] >= perFile {
			continue
		}
		count[h.Chunk.Path]++
		out = append(out, h)
		if len(out) >= k {
			break
		}
	}
	return out
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
