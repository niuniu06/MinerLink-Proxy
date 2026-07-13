package proxy

import (
	"sync"
	"testing"
	"time"
)

func TestStatsTracker_ConcurrentIncrements(t *testing.T) {
	st := NewStatsTracker()
	var wg sync.WaitGroup

	numGoroutines := 1000
	incrementsPerGoroutine := 1000

	// 1000 goroutines, each increments ValidShares and Shares 1000 times
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				st.IncValidShares()
				st.IncShares()
			}
		}()
	}

	wg.Wait()

	expected := uint64(numGoroutines * incrementsPerGoroutine)

	if st.ValidShares() != expected {
		t.Errorf("Expected ValidShares to be %d, got %d", expected, st.ValidShares())
	}

	if st.Shares() != expected {
		t.Errorf("Expected Shares to be %d, got %d", expected, st.Shares())
	}
}

func TestStatsTracker_Snapshot(t *testing.T) {
	st := NewStatsTracker()
	st.IncValidShares()
	st.IncValidShares()
	st.IncInvalidShares()
	st.IncFeeShares()

	snapshot := st.Snapshot()

	if snapshot.ValidShares != 2 {
		t.Errorf("Expected snapshot.ValidShares to be 2, got %d", snapshot.ValidShares)
	}
	if snapshot.InvalidShares != 1 {
		t.Errorf("Expected snapshot.InvalidShares to be 1, got %d", snapshot.InvalidShares)
	}
	if snapshot.FeeShares != 1 {
		t.Errorf("Expected snapshot.FeeShares to be 1, got %d", snapshot.FeeShares)
	}
	
	// Just ensure connected at was set within the last second
	if time.Since(snapshot.ConnectedAt) > time.Second {
		t.Errorf("Expected ConnectedAt to be recent, got %v", snapshot.ConnectedAt)
	}
}
