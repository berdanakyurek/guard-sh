package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultURL = "http://localhost:11434/api/chat"

// Client implements guard.Provider using the Ollama local API.
type Client struct {
	url   string
	model string
	http  *http.Client
}

func New(url, model string) *Client {
	if url == "" {
		url = defaultURL
	}
	return &Client{
		url:   url,
		model: model,
		http:  &http.Client{Timeout: 15 * time.Second},
	}
}

type requestBody struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  options   `json:"options"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type options struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
}

type responseBody struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

func (c *Client) Query(ctx context.Context, systemPrompt, command string) (string, error) {
	payload := requestBody{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: command},
		},
		Stream: false,
		Options: options{
			Temperature: 0,
			NumPredict:  150,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var result responseBody
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	text := strings.TrimSpace(result.Message.Content)
	if text == "" {
		return "OK", nil
	}
	return text, nil
}
