package answer

import (
	"fmt"
	"net/http"
	"encoding/json"
	"bytes"
)
type msg struct { Role, Content string }
type chatReq struct {
	Model string `json:"model"`
	Messages []msg `json:"messages"`
	Stream bool `json:"stream"`
}
type chatResp struct { Message msg}

func AskLLM(system, question, context string) (string, error) {
	msgs := []msg{
		{Role: "system", Content: system},
		{Role: "user", Content: fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context, question)},
	}
	b, _ := json.Marshal(chatReq{Model: "llama3.1:8b", Messages: msgs, Stream: false})
	resp, err := http.Post("http://localhost:11434/chat/completions", "application/json", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out chatResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Message.Content, nil
}
