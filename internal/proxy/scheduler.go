package proxy

import (
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

func isTimeInWindow(minute, start, end, cycle float64) bool {
	if end <= cycle {
		return minute >= start && minute < end
	}
	endWrapped := end - cycle
	return minute >= start || minute < endWrapped
}

func (s *FeeScheduler) scheduleMiners() {
	sessions := make([]*Session, 0)
	s.server.Sessions.Range(func(key, value interface{}) bool {
		if sess, ok := value.(*Session); ok {
			sessions = append(sessions, sess)
		}
		return true
	})

	// Sort sessions by Connection Time to ensure immutable scheduling order and prevent the "UUID Death Trap"
	sort.Slice(sessions, func(i, j int) bool {
		sessions[i].mu.Lock()
		tI := sessions[i].Stats.ConnectedAt().UnixNano()
		sessions[i].mu.Unlock()
		sessions[j].mu.Lock()
		tJ := sessions[j].Stats.ConnectedAt().UnixNano()
		sessions[j].mu.Unlock()

		// Fallback to ID if timestamps are identical to guarantee stable sort
		if tI == tJ {
			return sessions[i].ID < sessions[j].ID
		}
		return tI < tJ
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

		if devPercent <= 0 && opPercent <= 0 {
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

		opCycleMinsFloat := float64(config.FeeCycleMinutes)
		if opCycleMinsFloat <= 0 {
			opCycleMinsFloat = 100.0 // Default 100 minutes
		}
		devCycleMinsFloat := 100.0 // Hardcoded immutable cycle for developers

		nowUnix := float64(now.Unix()) / 60.0
		minuteInOpCycle := math.Mod(nowUnix, opCycleMinsFloat)
		minuteInDevCycle := math.Mod(nowUnix, devCycleMinsFloat)

		n := len(groupSessions)

		opSpacing := opCycleMinsFloat / float64(n)
		devSpacing := devCycleMinsFloat / float64(n)

		opDurationMins := opCycleMinsFloat * (opPercent / 100.0)
		devDurationMins := devCycleMinsFloat * (devPercent / 100.0)

		for i, sess := range groupSessions {
			// 1. Dev Timeline Check (Absolute Priority)
			isDevTime := false
			if devPercent > 0 {
				devStartMin := float64(i) * devSpacing
				devEndMin := devStartMin + devDurationMins
				isDevTime = isTimeInWindow(minuteInDevCycle, devStartMin, devEndMin, devCycleMinsFloat)
			}

			// 2. Operator Timeline Check
			isOpTime := false
			if opPercent > 0 {
				opStartMin := float64(i) * opSpacing
				opEndMin := opStartMin + opDurationMins
				isOpTime = isTimeInWindow(minuteInOpCycle, opStartMin, opEndMin, opCycleMinsFloat)
			}

			// Priority Arbiter: Dev always overrides Op during collisions
			targetMode := FeeModeNone
			if isDevTime {
				targetMode = FeeModeDev
			} else if isOpTime {
				targetMode = FeeModeOperator
			}

			sess.mu.Lock()
			currentMode := sess.CurrentFeeMode
			sess.mu.Unlock()

			if targetMode != FeeModeNone {
				if currentMode != targetMode {
					if targetMode == FeeModeDev {
						go sess.StartFeeMining(true)
					} else {
						go sess.StartFeeMining(false)
					}
				}
			} else {
				if currentMode != FeeModeNone {
					go sess.StopFeeMining()
				}
			}
		}
	}
}
