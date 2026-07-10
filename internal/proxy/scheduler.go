package proxy

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type FeeScheduler struct {
	server *Server
	mu     sync.Mutex
	quit   chan struct{}
}

func NewFeeScheduler(server *Server) *FeeScheduler {
	return &FeeScheduler{
		server: server,
		quit:   make(chan struct{}),
	}
}

func (s *FeeScheduler) Start() {
	go s.loop()
}

func (s *FeeScheduler) Stop() {
	close(s.quit)
}

func (s *FeeScheduler) loop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.scheduleMiners()
		}
	}
}

func (s *FeeScheduler) scheduleMiners() {
	sessions := make([]*Session, 0)
	s.server.Sessions.Range(func(key, value interface{}) bool {
		if sess, ok := value.(*Session); ok {
			sessions = append(sessions, sess)
		}
		return true
	})

	// Sort sessions by ID to ensure deterministic scheduling order
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})

	now := time.Now()
	// Group sessions by config to apply different schedules
	configGroups := make(map[string][]*Session)
	for _, sess := range sessions {
		sess.mu.Lock()
		config := sess.Config
		sess.mu.Unlock()
		if config != nil {
			configGroups[config.CoinName] = append(configGroups[config.CoinName], sess)
		}
	}

	for _, groupSessions := range configGroups {
		if len(groupSessions) == 0 {
			continue
		}
		
		config := groupSessions[0].Config
		devPercent := config.DevFeePercent
		opPercent := config.OperatorFeePercent
		totalFeePercent := devPercent + opPercent
		
		if totalFeePercent <= 0 {
			for _, sess := range groupSessions {
				sess.mu.Lock()
				currentMode := sess.CurrentFeeMode
				sess.mu.Unlock()
				if currentMode != FeeModeNone {
					go sess.StopFeeMining()
				}
			}
			continue
		}
		
		cycleMinsFloat := float64(config.FeeCycleMinutes)
		if cycleMinsFloat <= 0 {
			cycleMinsFloat = 100.0 // Default 100 minutes
		}
		
		// Ensure time cycle is aligned to epoch for consistency across restarts
		minuteInCycle := float64(now.Unix()) / 60.0
		minuteInCycle = math.Mod(minuteInCycle, cycleMinsFloat)
		
		n := len(groupSessions)
		
		spacing := cycleMinsFloat / float64(n)
		feeDurationMins := cycleMinsFloat * (totalFeePercent / 100.0)
		devDurationMins := cycleMinsFloat * (devPercent / 100.0)
		
		for i, sess := range groupSessions {
			sess.mu.Lock()
			isPRL := false
			if sess.Config != nil {
				isPRL = strings.ToUpper(sess.Config.CoinName) == "PRL"
			}
			sess.mu.Unlock()

			// Hard Isolation for PRL: Completely bypass the time-based scheduler
			if isPRL {
				continue
			}

			startMin := float64(i) * spacing
			endMin := startMin + feeDurationMins
			
			// Check if current minute falls into this miner's scheduled fee time
			isFeeTime := false
			targetMode := FeeModeNone
			
			if endMin <= cycleMinsFloat {
				if minuteInCycle >= startMin && minuteInCycle < endMin {
					isFeeTime = true
					if minuteInCycle < startMin + devDurationMins {
						targetMode = FeeModeDev
					} else {
						targetMode = FeeModeOperator
					}
				}
			} else {
				// Wraps around the cycle boundary (e.g. start at 99, end at 1)
				endMinWrapped := endMin - cycleMinsFloat
				if minuteInCycle >= startMin || minuteInCycle < endMinWrapped {
					isFeeTime = true
					
					// Determine Dev vs Op within wrapped boundary
					devEndMin := startMin + devDurationMins
					if devEndMin <= cycleMinsFloat {
						if minuteInCycle >= startMin && minuteInCycle < devEndMin {
							targetMode = FeeModeDev
						} else {
							targetMode = FeeModeOperator
						}
					} else {
						devEndMinWrapped := devEndMin - cycleMinsFloat
						if minuteInCycle >= startMin || minuteInCycle < devEndMinWrapped {
							targetMode = FeeModeDev
						} else {
							targetMode = FeeModeOperator
						}
					}
				}
			}
			
			sess.mu.Lock()
			currentMode := sess.CurrentFeeMode
			sess.mu.Unlock()
			
			if isFeeTime {
				if currentMode != targetMode {
					if targetMode == FeeModeDev {
						// Hide DEV fee logs from the system log
						// // removed scheduler log
						go sess.StartFeeMining(true)
					} else {
						// removed scheduler log
						go sess.StartFeeMining(false)
					}
				}
			} else {
				if currentMode != FeeModeNone {
					if currentMode != FeeModeDev {
						// removed scheduler log
					}
					go sess.StopFeeMining()
				}
			}
		}
	}
}
