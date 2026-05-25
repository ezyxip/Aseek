package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"aseek-orchestrator/internal/logging"
)

type Server struct {
	URL      string  `json:"url"`
	Weight   float64 `json:"weight"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type Profile struct {
	Name    string   `json:"name"`
	Servers []Server `json:"servers"`
	Prompt  string   `json:"prompt,omitempty"`
}

type Manager struct {
	path     string
	mu       sync.RWMutex
	all      []Profile
	active Profile
	log      *logging.Logger
}

func New(path string, log *logging.Logger) *Manager {
	return &Manager{
		path: path,
		log:  log.WithModule("profiles"),
	}
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("read profiles: %w", err)
	}

	var profiles []Profile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return fmt.Errorf("parse profiles: %w", err)
	}

	if len(profiles) == 0 {
		return fmt.Errorf("no profiles found")
	}

	m.all = profiles
	m.active = profiles[0]
	m.log.Info("profile loaded", "name", m.active.Name, "servers", len(m.active.Servers))
	return nil
}

func (m *Manager) GetServers() []Server {
	m.mu.RLock()
	defer m.mu.RUnlock()
	servers := make([]Server, len(m.active.Servers))
	copy(servers, m.active.Servers)
	return servers
}

func (m *Manager) ActiveProfile() Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

func (m *Manager) ActivePrompt() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active.Prompt
}

func (m *Manager) ListProfiles() []Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	profiles := make([]Profile, len(m.all))
	copy(profiles, m.all)
	return profiles
}

func (m *Manager) SwitchTo(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.all {
		if p.Name == name {
			m.active = p
			m.log.Info("switched to profile", "name", name, "servers", len(p.Servers))
			return nil
		}
	}

	return fmt.Errorf("profile not found: %s", name)
}