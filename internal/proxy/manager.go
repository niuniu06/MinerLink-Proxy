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

func (m *Manager) StartProxy(cfg models.ProxyConfig) error {
	// Stop existing if any
	m.StopProxy(cfg.ListenPort)

	// If the port is disabled, we just stop it and return
	if !cfg.Enabled {
		return nil
	}

	// Create new server
	// We need a pointer to the config
	cfgCopy := cfg
	server := NewServer(&cfgCopy)

	if err := server.Start(); err != nil {
		log.Printf("Failed to start proxy on port %d: %v", cfg.ListenPort, err)
		return err
	}

	m.Servers.Store(cfg.ListenPort, server)
	return nil
}

func (m *Manager) StopProxy(port int) {
	if s, ok := m.Servers.LoadAndDelete(port); ok {
		server := s.(*Server)
		server.Stop()
	}
}

func (m *Manager) RestartProxy(port int) error {
	// Load specific config from DB
	configs, _ := db.GetAllConfigs()
	for _, cfg := range configs {
		if cfg.ListenPort == port {
			return m.StartProxy(cfg)
		}
	}
	return nil
}

func (m *Manager) GetAllMiners() []map[string]interface{} {
	allMiners := make([]map[string]interface{}, 0)
	m.Servers.Range(func(key, value interface{}) bool {
		server := value.(*Server)
		_, miners := server.GetPaginatedMiners(1, 999999)
		for _, miner := range miners {
			// Add port info to the miner object
			miner["port"] = server.Config.ListenPort
			miner["coinName"] = server.Config.CoinName
			allMiners = append(allMiners, miner)
		}
		return true
	})
	return allMiners
}
