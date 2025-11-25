package retriever

import (
	"sort"
	"github.com/avh17/silo.git/internal/embed"
	"github.com/avh17/silo.git/internal/store"
)

type Ranked struct {
	Chunk store.Chunk
	Score float64
}

func Search(db *store.DB, query string, k int) ([]Ranked, error) {
	ids, bm, err := store.SearchBM25(db, query, k)
	if err != nil {return nil, err}

	embQ, err := embed.Embed(query)
	if err != nil {return nil, err}

	chunkMap, err := store.GetChunksByIDs(db, ids)
	if err != nil {return nil, err}

	type row struct {
		id int64
		score float64
	}
	var rows []row
	for _, id := range ids {
		vec, err := store.GetEmbedding(db, id)
		if err != nil { continue }
		cos := Cosine(embQ, vec)
		score := 0.5*bm[id] + 0.5*cos
		rows = append(rows, row{id: id, score: score})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].score > rows[j].score})
	if len(rows) > k {rows = rows[:k]}

	out := make([]Ranked, 0, len(rows))
	for _,r := range rows {
		out = append(out, Ranked{Chunk: chunkMap[r.id], Score: r.score})
	}
	return out, nil
}