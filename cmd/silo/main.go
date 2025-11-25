package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"path/filepath"

	"github.com/avh17/silo.git/internal/answer"
	"github.com/avh17/silo.git/internal/embed"
	"github.com/avh17/silo.git/internal/retriever"
	"github.com/avh17/silo.git/internal/store"
	"github.com/avh17/silo.git/internal/parse"
	"github.com/avh17/silo.git/internal/chunk"
	"github.com/avh17/silo.git/internal/provenance"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: silo [index <path> | ask <question>]")
		os.Exit(1)
	}

	db, err := store.Open("silo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := store.Init(db); err != nil { log.Fatal(err) }

	switch os.Args[1] {
	case "index":
		if len(os.Args) != 3 {
			log.Fatal("Usage: silo index <folder>")
		}
		path := os.Args[2]
		if  err := indexPath(db, path); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Index complete")
	case "ask":
		if len(os.Args) != 3 {
			log.Fatal("Usage: silo ask <question>")
		}
		question := strings.Join(os.Args[2:], " ")
		if err := ask(db, question); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func indexPath(db *store.DB, root string) error {
	ctx := context.Background()
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() { return err }
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".pdf" || ext == ".md" || ext == ".markdown" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil { return err }
	
	for _, file := range files {
		fmt.Println("Parsing:", file)
		doc, err := parse.ParseFile(ctx, file)
		if err != nil { return err }

		chunks := chunk.MakeChunks(doc, 2500)
		for _, c := range chunks {
			id, err := store.InsertChunk(db, c)
			if err != nil { return err }
			vec, err := embed.Embed(c.Text)
			if err != nil { return err }
			if err := store.UpsertEmbedding(db, id, vec); err != nil { return err }
		}
	}
	return nil
}

func ask(db *store.DB, question string) error {
	top, err := retriever.Search(db, question, 8)
	if err != nil { return err }
	
	var ctxParts []string
	for _, t := range top {
		ctxParts = append(ctxParts, fmt.Sprintf("[[ %s: %d ]] %s", t.Chunk.DocPath, t.Chunk.Page, t.Chunk.Text))
	}
	ctxStr := strings.Join(ctxParts, "\n---\n")
	
	system :=  "You answer strictly from the provided context. Be concise. If unsure, say so."
	ans, err:= answer.AskLLM(system, question, ctxStr)
	if err != nil { return err }
	
	sents := provenance.SplitSentences(ans)
	prov, err := provenance.MapSentences(db, sents, top)
	if err != nil { return err }
	
	fmt.Println("\nAnswer:\n", ans, "\n")
	fmt.Println("Citations:")
	for i, p := range prov {
		fmt.Printf(" (%d) %s [%s p.%d]\n", i+1, p.Sentence, p.DocPath, p.Page)
	}
	return nil
}