package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aseek-orchestrator/internal/logging"
	"aseek-orchestrator/internal/profile"
)

type searchRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

type searchResponse struct {
	Results []Document `json:"results"`
}

type Document struct {
	Index   int     `json:"index"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Source  string  `json:"source"`
	Score   float64 `json:"score"`
}

type Pipeline struct {
	profiles       *profile.Manager
	log            *logging.Logger
	defaultTimeout time.Duration
	rerankerURL    string
}

func New(defaultTimeout time.Duration, profiles *profile.Manager, log *logging.Logger, rerankerURL string) *Pipeline {
	return &Pipeline{
		profiles:       profiles,
		log:            log.WithModule("pipeline"),
		defaultTimeout: defaultTimeout,
		rerankerURL:    rerankerURL,
	}
}

func (p *Pipeline) Execute(ctx context.Context, query string) ([]Document, error) {
	servers := p.profiles.GetServers()
	if len(servers) == 0 {
		return nil, fmt.Errorf("no search servers configured")
	}

	var allDocs []Document
	var lastErr error
	seenErrors := 0

	for _, srv := range servers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		docs, err := p.searchServer(ctx, srv, query)
		if err != nil {
			p.log.Warn("search server failed", "server", srv.URL, "error", err)
			seenErrors++
			lastErr = err
			continue
		}
		allDocs = append(allDocs, docs...)
	}

	if len(allDocs) == 0 && seenErrors == len(servers) {
		return nil, fmt.Errorf("all search servers failed: %w", lastErr)
	}

	allDocs = p.rerank(ctx, query, allDocs)

	return allDocs, nil
}

func (p *Pipeline) searchServer(ctx context.Context, srv profile.Server, query string) ([]Document, error) {
	timeout := p.defaultTimeout
	if srv.TimeoutMs > 0 {
		timeout = time.Duration(srv.TimeoutMs) * time.Millisecond
	}
	client := &http.Client{Timeout: timeout}

	topK := 5
	reqBody := searchRequest{Query: query, TopK: topK}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal search request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/api/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search server returned %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	return sr.Results, nil
}

type rerankRequest struct {
	Query string   `json:"query"`
	Texts []string `json:"texts"`
}

type rerankResponse struct {
	Results []rerankResult `json:"results"`
}

type rerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

func (p *Pipeline) rerank(ctx context.Context, query string, docs []Document) []Document {
	if p.rerankerURL == "" || len(docs) == 0 {
		return docs
	}

	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.Content
	}

	reqBody := rerankRequest{Query: query, Texts: texts}
	body, err := json.Marshal(reqBody)
	if err != nil {
		p.log.Warn("rerank marshal", "error", err)
		return docs
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.rerankerURL+"/reranking", bytes.NewReader(body))
	if err != nil {
		p.log.Warn("rerank request", "error", err)
		return docs
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: p.defaultTimeout}
	resp, err := client.Do(req)
	if err != nil {
		p.log.Warn("rerank call", "error", err)
		return docs
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.log.Warn("rerank status", "code", resp.StatusCode)
		return docs
	}

	var rr rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		p.log.Warn("rerank decode", "error", err)
		return docs
	}

	ordered := make([]Document, len(docs))
	for i, r := range rr.Results {
		if r.Index >= 0 && r.Index < len(docs) {
			d := docs[r.Index]
			d.Score = r.Score
			ordered[i] = d
		}
	}

	return ordered
}