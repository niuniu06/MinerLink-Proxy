package proxy

import (
	"log"
	"sync"

	"proxy-core/internal/db"
	"proxy-core/internal/models"
)

type Manager struct {
	Servers sync.Map // map[int]*Server (key is ListenPort)
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) LoadAllAndStart() {
	configs, err := db.GetAllConfigs()
	if err != nil {
		log.Printf("Failed to load configs from DB: %v", err)
		return
	}

	for _, cfg := range configs {
		m.StartProxy(cfg)
	}
}

func (m *Manager) StartProxy(cfg models.ProxyConfig) {
	// Stop existing if any
	m.StopProxy(cfg.ListenPort)

	// Create new server
	// We need a pointer to the config
	cfgCopy := cfg
	server := NewServer(&cfgCopy)
	m.Servers.Store(cfg.ListenPort, server)

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Failed to start proxy on port %d: %v", cfg.ListenPort, err)
			m.Servers.Delete(cfg.ListenPort)
		}
	}()
}

func (m *Manager) StopProxy(port int) {
	if s, ok := m.Servers.LoadAndDelete(port); ok {
		server := s.(*Server)
		server.Stop()
	}
}

func (m *Manager) RestartProxy(port int) {
	// Load specific config from DB
	configs, _ := db.GetAllConfigs()
	for _, cfg := range configs {
		if cfg.ListenPort == port {
			m.StartProxy(cfg)
			break
		}
	}
}
