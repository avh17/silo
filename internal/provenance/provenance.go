package provenance

import (
	"regexp"
	"strings"

	"github.com/avh17/silo.git/internal/retriever"
	"github.com/avh17/silo.git/internal/store"
	"github.com/avh17/silo.git/internal/embed"
	
)
type SentenceProv struct {
	Sentence string
	DocPath string
	Page int
	Start int
	End int
}

var splitter = regexp.MustCompile(`(?m)([^.!?]+[.!?])`)

func SplitSentences(text string) []string{
	m := splitter.FindAllString(text, -1)
	if len(m) == 0 {
		t := strings.TrimSpace(text)
		if t == "" { return nil }
		return []string{t}
	}
	for i := range m { m[i] = strings.TrimSpace(m[i]) }
	return m
}

func MapSentences(db *store.DB, sents []string, candidates []retriever.Ranked) ([]SentenceProv, error) {
	var out []SentenceProv
	for _, s := range sents {
		emb, err := embed.Embed(s)
		if err != nil {return nil, err}
		var best struct {
			score float64
			ch store.Chunk
		}
		for _, c := range candidates {
			vec, err := store.GetEmbedding(db, c.Chunk.ID)
			if err != nil {continue}

			score := retriever.Cosine(emb, vec)
			if score > best.score {
				best.score = score
				best.ch = c.Chunk
			}
		}
		out = append(out, SentenceProv{
			Sentence: s, DocPath: best.ch.DocPath, Page: best.ch.Page, Start: best.ch.Start, End: best.ch.End,
		})
	}
	return out, nil
}