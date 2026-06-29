package proxy

import (
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

type FeeScheduler struct {
	server *Server
	mu     sync.Mutex
	quit   chan struct{}
}

func NewFeeScheduler(s *Server) *FeeScheduler {
	return &FeeScheduler{
		server: s,
		quit:   make(chan struct{}),
	}
}

func (fs *FeeScheduler) Start() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	log.Printf("Starting Stateless Distributed Fee Scheduler...")

	for {
		select {
		case <-fs.quit:
			return
		case <-ticker.C:
			fs.processTick()
		}
	}
}

func (fs *FeeScheduler) Stop() {
	close(fs.quit)
}

func (fs *FeeScheduler) processTick() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 1. Snapshot active sessions by wallet
	sessionsByWallet := make(map[string][]*Session)
	
	fs.server.Sessions.Range(func(key, value interface{}) bool {
		sess := value.(*Session)
		sess.mu.Lock()
		isOffline := sess.IsOffline
		wallet := sess.MinerWallet
		sess.mu.Unlock()
		
		if !isOffline && wallet != "" {
			sessionsByWallet[wallet] = append(sessionsByWallet[wallet], sess)
		}
		return true
	})

	devPercent := fs.server.Config.DevFeePercent
	opPercent := fs.server.Config.OperatorFeePercent
	totalFeePercent := devPercent + opPercent
	
	if totalFeePercent <= 0 {
		return
	}

	cycleMins := fs.server.Config.FeeCycleMinutes
	if cycleMins <= 0 {
		cycleMins = 100 // Default to 100 minutes
	}
	
	cycleMinsFloat := float64(cycleMins)
	
	// Determine current minute inside the cycle
	nowUnix := time.Now().Unix()
	currentMinute := float64(nowUnix / 60)
	minuteInCycle := math.Mod(currentMinute, cycleMinsFloat)

	// 2. Schedule each wallet
	for _, sessions := range sessionsByWallet {
		n := len(sessions)
		if n == 0 {
			continue
		}
		
		// Sort sessions by ID to ensure a stable, deterministic order
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].ID < sessions[j].ID
		})
		
		spacing := cycleMinsFloat / float64(n)
		feeDurationMins := cycleMinsFloat * (totalFeePercent / 100.0)
		devDurationMins := cycleMinsFloat * (devPercent / 100.0)
		
		for i, sess := range sessions {
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
						// log.Printf("[Scheduler] Miner %s entering DEV fee time slot", sess.MinerWorker)
						go sess.StartFeeMining(true)
					} else {
						log.Printf("[Scheduler] Miner %s entering OP fee time slot", sess.MinerWorker)
						go sess.StartFeeMining(false)
					}
				}
			} else {
				if currentMode != FeeModeNone {
					if currentMode != FeeModeDev {
						log.Printf("[Scheduler] Miner %s finishing fee time slot, returning to Main", sess.MinerWorker)
					}
					go sess.StopFeeMining()
				}
			}
		}
	}
}
