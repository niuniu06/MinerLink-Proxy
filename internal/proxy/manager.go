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

// GetMinerHistory dynamically generates a 72-hour hashrate curve from in-memory ring buffers
func (m *Manager) GetMinerHistory(ip string) models.HistoryResponse {
	history := make([]models.HashrateHistory, 0)
	summary := make(map[string]models.HashrateSummary)
	m.Servers.Range(func(key, value interface{}) bool {
		server := value.(*Server)
		server.Sessions.Range(func(k, v interface{}) bool {
			sess := v.(*Session)
			if sess.ID == ip {
				// Found the session
				sess.RingBuffer.mu.RLock()
				idx := sess.RingBuffer.CurrentIndex
				
				now := time.Now()
				
				// Compute real-time summaries before we group by hour
				m10, f10 := sess.RingBuffer.GetAvgHashrate(10)
				m1h, f1h := sess.RingBuffer.GetAvgHashrate(60)
				m6h, f6h := sess.RingBuffer.GetAvgHashrate(360)
				
				baseMHs := sess.getAlgoBaseMHs()
				summary["avg10m"] = models.HashrateSummary{
					MainHashrate: (m10 * baseMHs) / (10 * 60.0),
					FeeHashrate:  (f10 * baseMHs) / (10 * 60.0),
				}
				summary["avg1h"] = models.HashrateSummary{
					MainHashrate: (m1h * baseMHs) / (60 * 60.0),
					FeeHashrate:  (f1h * baseMHs) / (60 * 60.0),
				}
				summary["avg6h"] = models.HashrateSummary{
					MainHashrate: (m6h * baseMHs) / (360 * 60.0),
					FeeHashrate:  (f6h * baseMHs) / (360 * 60.0),
				}

				tHour := now.Truncate(time.Hour)
				minutesIterated := 0
				
				for h := 0; h < 72; h++ {
					var sumMain, sumFee float64
					var minutesInBlock int
					
					if h == 0 {
						minutesInBlock = now.Minute() + 1
					} else {
						minutesInBlock = 60
					}
					
					for m := 0; m < minutesInBlock; m++ {
						bucketIdx := idx - minutesIterated
						for bucketIdx < 0 {
							bucketIdx += 4320
						}
						
						sumMain += sess.RingBuffer.MainHash[bucketIdx]
						sumFee += sess.RingBuffer.FeeHash[bucketIdx]
						
						minutesIterated++
					}
					
					mainH := (sumMain * baseMHs) / float64(minutesInBlock*60)
					feeH := (sumFee * baseMHs) / float64(minutesInBlock*60)
					
					history = append(history, models.HashrateHistory{
						Timestamp:    tHour,
						MainHashrate: mainH,
						FeeHashrate:  feeH,
					})
					
					tHour = tHour.Add(-time.Hour)
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
	
	return models.HistoryResponse{
		Summary: summary,
		History: history,
	}
}

