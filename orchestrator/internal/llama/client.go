package llama

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"aseek-orchestrator/internal/logging"
)

type Client struct {
	baseURL string
	client  *http.Client
	log     *logging.Logger
}

func New(baseURL string, client *http.Client, log *logging.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		client:  client,
		log:     log.WithModule("llama"),
	}
}

func (c *Client) Generate(ctx context.Context, systemPrompt, ragPrompt string, tokenCh chan<- string) error {
	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": ragPrompt},
	}

	body, err := json.Marshal(map[string]interface{}{
		"messages":    messages,
		"stream":      true,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("llama-server returned %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			return nil
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			c.log.Warn("parse SSE chunk", "error", err)
			continue
		}

		if len(chunk.Choices) > 0 {
			token := chunk.Choices[0].Delta.Content
			if token != "" {
				select {
				case tokenCh <- token:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}

	return scanner.Err()
}