package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LlamaConfig struct {
	Binary      string  `json:"binary"`
	Model       string  `json:"model"`
	Port        int     `json:"port"`
	CtxSize     int     `json:"ctx_size"`
	Threads     int     `json:"threads"`
	Slots       int     `json:"slots"`
	Batch       int     `json:"batch"`
	GPULayers   int     `json:"gpu_layers"`
	Temperature float64 `json:"temperature"`
}

type StreamingConfig struct {
	FlushIntervalMs int `json:"flush_interval_ms"`
}

type NetworkConfig struct {
	RequestTimeoutMs int `json:"request_timeout_ms"`
}

type LoggingConfig struct {
	Level string `json:"level"`
}

type RerankerConfig struct {
	URL string `json:"url"`
}

type Config struct {
	Llama     LlamaConfig     `json:"llama"`
	Streaming StreamingConfig `json:"streaming"`
	Network   NetworkConfig   `json:"network"`
	Logging   LoggingConfig   `json:"logging"`
	Reranker  RerankerConfig  `json:"reranker"`
}

func (c *Config) FlushInterval() time.Duration {
	return time.Duration(c.Streaming.FlushIntervalMs) * time.Millisecond
}

func (c *Config) RequestTimeout() time.Duration {
	return time.Duration(c.Network.RequestTimeoutMs) * time.Millisecond
}

type Manager struct {
	path string
}

func NewManager(path string) *Manager {
	return &Manager{path: path}
}

func (m *Manager) Load() (*Config, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Llama.Binary == "" {
		return nil, fmt.Errorf("config: llama.binary is required")
	}
	if cfg.Llama.Model == "" {
		return nil, fmt.Errorf("config: llama.model is required")
	}
	if cfg.Llama.Port <= 0 || cfg.Llama.Port > 65535 {
		return nil, fmt.Errorf("config: llama.port must be 1-65535, got %d", cfg.Llama.Port)
	}

	return &cfg, nil
}

func DefaultPath() string {
	if v := os.Getenv("AURORA_CONFIG"); v != "" {
		return v
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "aurora-rag", "orchestrator.json")
}

func DefaultProfilesPath() string {
	if v := os.Getenv("AURORA_PROFILES"); v != "" {
		return v
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "aurora-rag", "profiles.json")
}

func DefaultTemplatesDir() string {
	if v := os.Getenv("AURORA_TEMPLATES"); v != "" {
		return v
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "aurora-rag", "prompts")
}