package proxy

import (
	"log"
	"sync"
	"time"

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

func (m *Manager) GetAllMiners() []GlobalMinerStats {
	allMiners := make([]GlobalMinerStats, 0)
	m.Servers.Range(func(key, value interface{}) bool {
		server := value.(*Server)
		_, miners := server.GetPaginatedMiners()
		for _, miner := range miners {
			globalMiner := GlobalMinerStats{
				MinerStatsData: miner,
				Port:           server.Config.ListenPort,
				CoinName:       server.Config.CoinName,
			}
			allMiners = append(allMiners, globalMiner)
		}
		return true
	})
	return allMiners
}

// GetMinerHistory dynamically generates a 24-hour hashrate curve from in-memory ring buffers
func (m *Manager) GetMinerHistory(ip string) []models.HashrateHistory {
	history := make([]models.HashrateHistory, 0)
	m.Servers.Range(func(key, value interface{}) bool {
		server := value.(*Server)
		server.Sessions.Range(func(k, v interface{}) bool {
			sess := v.(*Session)
			if sess.ID == ip {
				// Found the session
				sess.RingBuffer.mu.RLock()
				idx := sess.RingBuffer.CurrentIndex
				
				// Reconstruct time series for the last 360 minutes (6 hours)
				// Or fewer if it hasn't been online that long, but we just emit the whole buffer
				// because empty minutes will be 0. We'll emit 144 points (every 2.5 mins on average)
				// But let's just emit up to 360 points (every minute)
				
				now := time.Now()
				for i := 0; i < 360; i++ {
					t := now.Add(-time.Duration(i) * time.Minute)
					bucketIdx := (idx - i)
					if bucketIdx < 0 {
						bucketIdx += 360
					}
					
					mainH := sess.RingBuffer.MainHash[bucketIdx] * sess.getAlgoBaseMHs() / 60.0
					feeH := sess.RingBuffer.FeeHash[bucketIdx] * sess.getAlgoBaseMHs() / 60.0
					
					// We only append if there's hashrate to keep payload small, or we just append all
					history = append(history, models.HashrateHistory{
						Timestamp:    t,
						MainHashrate: mainH,
						FeeHashrate:  feeH,
					})
				}
				sess.RingBuffer.mu.RUnlock()
				return false // Stop iterating sessions in this server
			}
			return true
		})
		if len(history) > 0 {
			return false // Stop iterating servers
		}
		return true
	})
	
	// Reverse history so it is chronological
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}
	
	return history
}

