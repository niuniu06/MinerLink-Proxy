package proxy

import (
	"fmt"
	"log"
	"math"
	"time"
)

// StartVardiffEngine starts a background goroutine to periodically check and adjust local difficulty.
func (s *Session) StartVardiffEngine() {
	if !s.Config.EnableVardiff || s.Config.TargetShareRate <= 0 {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial setup: Wait for initial RemoteDiff
	time.Sleep(5 * time.Second)
	s.mu.Lock()
	if s.LocalDiff == 0 {
		if s.RemoteDiff > 0 {
			s.LocalDiff = s.RemoteDiff
		} else {
			s.LocalDiff = 1000 // Fallback starting diff
		}
	}
	s.mu.Unlock()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.evaluateVardiff()
		}
	}
}

func (s *Session) evaluateVardiff() {
	var setDiffPkt string
	var minerConn net.Conn

	func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if s.MinerConn == nil || s.State != "MAIN" {
			return
		}

		// Count shares in the last minute
		now := time.Now()
		cutoff := now.Add(-1 * time.Minute)
		sharesLastMinute := 0
		for _, ev := range s.ShareHistory {
			if ev.Timestamp.After(cutoff) {
				sharesLastMinute++
			}
		}

		target := s.Config.TargetShareRate
		if target <= 0 {
			return
		}

		// Tolerance range: +/- 30% of target
		lowerBound := int(float64(target) * 0.7)
		upperBound := int(float64(target) * 1.3)

		oldDiff := s.LocalDiff
		var newDiff float64 = oldDiff

		if sharesLastMinute > upperBound {
			// Too many shares, increase difficulty
			ratio := float64(sharesLastMinute) / float64(target)
			// Cap max adjustment to 4x to prevent wild swings
			if ratio > 4.0 {
				ratio = 4.0
			}
			newDiff = oldDiff * ratio
		} else if sharesLastMinute < lowerBound {
			// Too few shares, decrease difficulty
			ratio := float64(sharesLastMinute) / float64(target)
			// If 0 shares, forcefully cut in half
			if sharesLastMinute == 0 {
				newDiff = oldDiff / 2.0
			} else {
				if ratio < 0.25 {
					ratio = 0.25
				}
				newDiff = oldDiff * ratio
			}
		}

		if newDiff != oldDiff {
			// Round to nearest integer for cleanliness (many miners don't like deep decimals)
			newDiff = math.Round(newDiff)
			if newDiff <= 0 {
				newDiff = 1
			}
			s.LocalDiff = newDiff
			log.Printf("[Vardiff] Miner %s rate=%d/min. Adjusting LocalDiff %.0f -> %.0f", s.ID, sharesLastMinute, oldDiff, newDiff)

			// Send new difficulty to miner
			setDiffPkt = fmt.Sprintf(`{"id": null, "method": "mining.set_difficulty", "params": [%.0f]}`+"\n", newDiff)
			minerConn = s.MinerConn
		}
	}()

	if setDiffPkt != "" && minerConn != nil {
		fmt.Fprintf(minerConn, "%s", setDiffPkt)
	}
}
