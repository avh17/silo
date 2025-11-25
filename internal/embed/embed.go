
package embed

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type embedReq struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResp struct {
	Embedding []float32 `json:"embedding"`
}

func Embed(text string) ([]float32, error) {
	body, _ := json.Marshal(embedReq{
		Model: "nomic-embed-text",
		Input: text,
	})
	resp, err := http.Post("http://localhost:11434/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
    var out embedResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Embedding, nil
}
