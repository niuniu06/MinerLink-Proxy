package proxy

import (
	"sync"
	"time"

	"proxy-core/internal/db"
	"proxy-core/internal/models"
)

// HashrateRingBuffer stores historical hashrate per minute for the last 72 hours (4320 minutes)
type HashrateRingBuffer struct {
	mu           sync.RWMutex
	MainHash     [4320]float64
	FeeHash      [4320]float64
	CurrentIndex int
}

// AddShare adds difficulty to the current minute's bucket
func (r *HashrateRingBuffer) AddShare(diff float64, isFee bool, isDevFee bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if isFee && !isDevFee {
		r.FeeHash[r.CurrentIndex] += diff
	} else if !isFee {
		r.MainHash[r.CurrentIndex] += diff
	}
}

// Tick moves the ring buffer forward by one minute and clears the new bucket
func (r *HashrateRingBuffer) Tick() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.CurrentIndex = (r.CurrentIndex + 1) % 4320
	r.MainHash[r.CurrentIndex] = 0
	r.FeeHash[r.CurrentIndex] = 0
}

// GetAvgHashrate calculates the average hashrate over the last `minutes` (up to 4320).
// Returns (mainHashrate, feeHashrate).
// The passed formula multiplier should be applied by the caller (e.g. diff * 2^32 / (minutes * 60)).
func (r *HashrateRingBuffer) GetAvgHashrate(minutes int) (float64, float64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if minutes > 4320 {
		minutes = 4320
	}
	if minutes <= 0 {
		return 0, 0
	}

	var totalMain, totalFee float64
	idx := r.CurrentIndex

	for i := 0; i < minutes; i++ {
		totalMain += r.MainHash[idx]
		totalFee += r.FeeHash[idx]
		idx--
		if idx < 0 {
			idx = 4319
		}
	}

	return totalMain, totalFee
}

// FlushToDB saves the current snapshot to SQLite. 
// It calculates the average hashrate over the flush interval (e.g., last 30 minutes).
func (r *HashrateRingBuffer) FlushToDB(port int, coinName string, intervalMinutes int, diffMultiplier float64) {
	mainSum, feeSum := r.GetAvgHashrate(intervalMinutes)
	// Calculate H/s
	timeSeconds := float64(intervalMinutes * 60)
	mainHs := (mainSum * diffMultiplier) / timeSeconds
	feeHs := (feeSum * diffMultiplier) / timeSeconds

	db.DB.Create(&models.HashrateHistory{
		Timestamp:    time.Now(),
		Port:         port,
		CoinName:     coinName,
		MainHashrate: mainHs,
		FeeHashrate:  feeHs,
	})
}
