package chunk

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"github.com/avh17/silo.git/internal/store"
	"github.com/avh17/silo.git/internal/parse"
)


func MakeChunks(doc parse.Document, approxSize int) []store.Chunk {
	text := strings.TrimSpace(doc.Text)
	paras := strings.Split(text, "\n\n")
	var out []store.Chunk
	var cur strings.Builder
	start := 0
	for _,p := range paras {
		toAdd := p
		if cur.Len()+len(toAdd) > approxSize && cur.Len() > 0 {
			chText := cur.String()
			hash := sha256.Sum256([]byte(chText))
			out = append(out, store.Chunk{
				DocPath: doc.Path, Page: 1, Start: start, End: start+utf8.RuneCountInString(chText),
				SHA256: hex.EncodeToString(hash[:]), Text: chText,
			})
			start += utf8.RuneCountInString(chText)
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
		}
		cur.WriteString(toAdd)
	}
	if cur.Len() > 0 {
		chText := cur.String()
		hash := sha256.Sum256([]byte(chText))
		out = append(out, store.Chunk{
			DocPath: doc.Path, Page: 1, Start: start, End: start+utf8.RuneCountInString(chText),
			SHA256: hex.EncodeToString(hash[:]), Text: chText,
		})
	}
	return  out
}
	
