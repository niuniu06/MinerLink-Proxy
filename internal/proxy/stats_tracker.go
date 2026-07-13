package proxy

import (
	"sync/atomic"
	"time"
)

// StatsTracker encapsulates all session statistics and provides lock-free, concurrent updates.
// It is specifically designed to prevent state corruption under high concurrency (e.g. 10,000+ miners).
type StatsTracker struct {
	shares        uint64
	feeShares     uint64
	validShares   uint64
	invalidShares uint64
	connectedAt   int64 // Unix nano
}

func NewStatsTracker() *StatsTracker {
	return &StatsTracker{
		connectedAt: time.Now().UnixNano(),
	}
}

// Increment methods (Lock-free using atomic operations)
func (st *StatsTracker) IncShares() {
	atomic.AddUint64(&st.shares, 1)
}

func (st *StatsTracker) IncFeeShares() {
	atomic.AddUint64(&st.feeShares, 1)
}

func (st *StatsTracker) IncValidShares() {
	atomic.AddUint64(&st.validShares, 1)
}

func (st *StatsTracker) IncInvalidShares() {
	atomic.AddUint64(&st.invalidShares, 1)
}

// Getter methods (Lock-free atomic reads)
func (st *StatsTracker) Shares() uint64 {
	return atomic.LoadUint64(&st.shares)
}

func (st *StatsTracker) FeeShares() uint64 {
	return atomic.LoadUint64(&st.feeShares)
}

func (st *StatsTracker) ValidShares() uint64 {
	return atomic.LoadUint64(&st.validShares)
}

func (st *StatsTracker) InvalidShares() uint64 {
	return atomic.LoadUint64(&st.invalidShares)
}

func (st *StatsTracker) ConnectedAt() time.Time {
	return time.Unix(0, atomic.LoadInt64(&st.connectedAt))
}

func (st *StatsTracker) SetConnectedAt(t time.Time) {
	atomic.StoreInt64(&st.connectedAt, t.UnixNano())
}

// SessionStats is used for API JSON serialization or snapshotting state
type SessionStats struct {
	Shares        int64     `json:"shares"`
	FeeShares     int64     `json:"feeShares"`
	ValidShares   int64     `json:"validShares"`
	InvalidShares int64     `json:"invalidShares"`
	ConnectedAt   time.Time `json:"connectedAt"`
}

// Snapshot takes a frozen atomic read of the current statistics
func (st *StatsTracker) Snapshot() SessionStats {
	return SessionStats{
		Shares:        int64(st.Shares()),
		FeeShares:     int64(st.FeeShares()),
		ValidShares:   int64(st.ValidShares()),
		InvalidShares: int64(st.InvalidShares()),
		ConnectedAt:   st.ConnectedAt(),
	}
}
