package store

import (
	"bytes"
	"database/sql"
	"time"
	"encoding/binary"
	"errors"
	
	_ "modernc.org/sqlite"
)

type DB struct{ *sql.DB }

type Chunk struct {
	ID int64
	DocPath string
	Page int
	Start int
	End int
	SHA256 string
	Text string
	Created time.Time
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil { return nil, err }
	db.Exec("PRAGMA journal_mode = WAL;")
	db.Exec("PRAGMA synchronous = NORMAL;")
	return &DB{db}, nil
}

func Init(db *DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY,
			doc_path TEXT,
			page INT,
			start INT,
			end INT,
			sha256 TEXT,
			text TEXT,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(text, content='chunks', content_rowid='id');
		CREATE TABLE IF NOT EXISTS chunk_emb(
			chunk_id INTEGER PRIMARY KEY,
			dim INT NOT NULL,
			vec BLOB NOT NULL,
		);
        CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
			INSERT INTO chunks_fts(rowid, text) VALUES (new.id, new.text);
		END;
		CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
		 	INSERT INTO chunks_fts(chunks_fts, rowid, text) VALUES ('delete', old.id, old.text);
		END;
		CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
		 	INSERT INTO chunks_fts(chunks_fts, rowid, text) VALUES ('delete', old.id, old.text);
			INSERT INTO chunks_fts VALUES (new.id, new.text);
		END;
	`)
	return err
}

func InsertChunk(db *DB, c Chunk) (int64, error) {
	res, err := db.Exec("INSERT INTO chunks (doc_path, page, start, end, sha256, text) VALUES (?, ?, ?, ?, ?, ?)", c.DocPath, c.Page, c.Start, c.End, c.SHA256, c.Text)
	if err != nil { return 0, err }
	return res.LastInsertId()
}

func UpsertEmbedding(db *DB, chunkID int64, vec []float32) error {
	if len(vec) == 0 { return errors.New("empty vector") }
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, vec)
	_, err := db.Exec("INSERT OR REPLACE INTO chunk_emb (chunk_id, dim, vec) VALUES (?, ?, ?) ON CONFLICT(chunk_id) DO UPDATE SET dim = excluded.dim, vec = excluded.vec", chunkID, len(vec), b.Bytes())
	return err
}

func GetEmbedding(db *DB, chunkID int64) ([]float32, error) {
	var dim int
	var blob []byte
	if err := db.QueryRow("SELECT dim, vec FROM chunk_emb WHERE chunk_id = ?", chunkID).Scan(&dim, &blob); err != nil {
		return nil, err
	}

	out := make([]float32, dim)
	_ = binary.Read(bytes.NewReader(blob), binary.LittleEndian, &out)
	return out, nil
}

func GetChunksByIDs(db *DB, ids []int64) (map[int64]Chunk, error) {
    if len(ids) == 0 { return map[int64]Chunk{}, nil}
	q := "SELECT rowid, doc_path, page, start, end, sha256, text FROM chunks WHERE id IN (" + placeholders(len(ids)) + ")"
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := db.Query(q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	
	res := map[int64]Chunk{}
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.DocPath, &c.Page, &c.Start, &c.End, &c.SHA256, &c.Text); err != nil { return nil, err }
		res[c.ID] = c
	}
	return res, nil

}

func SearchBM25(db *DB, q string, k int) ([]int64, map[int64]float64, error){
	rows, err := db.Query("SELECT rowid, bm25(chunk_fts) AS score from chunk_fts where chunk_fts MATCH ? ORDER BY score LIMIT ?", q, k*4)
	if err != nil { return nil, nil, err }
	defer rows.Close()
	
	var	ids []int64
	scores := map[int64]float64{}
	for rows.Next() {
		var id int64
		var score float64
		if err := rows.Scan(&id, &score); err != nil { return nil, nil, err }
		ids = append(ids, id)
		scores[id] = -score // lower bm25() is better rank
	}
	return ids, scores, nil
}	

func placeholders(n int) string {
	if n<=0 { return "" }
	s := "?"
	for i := 1; i < n; i++ {
		s += ", ?"
	}
	return s
}

	