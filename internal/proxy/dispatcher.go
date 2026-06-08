package proxy

import (
	"sync"
)

// JobDispatcher provides a global caching mechanism for Stratum jobs and difficulties.
// This allows zero-latency job injections when miners switch states,
// and significantly reduces stale shares by fan-out broadcasting new difficulties and targets.
type JobDispatcher struct {
	mu           sync.RWMutex
	latestJobs   map[string]string   // Key: pool address, Value: raw mining.notify JSON
	latestDiff   map[string]float64  // Key: pool address, Value: latest difficulty
	subscribers  map[string][]chan string
}

var GlobalDispatcher = &JobDispatcher{
	latestJobs:  make(map[string]string),
	latestDiff:  make(map[string]float64),
	subscribers: make(map[string][]chan string),
}

func (d *JobDispatcher) UpdateJob(pool string, jobJSON string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.latestJobs[pool] = jobJSON
	
	// Fan-out broadcast to all subscribers of this pool
	if subs, exists := d.subscribers[pool]; exists {
		for _, ch := range subs {
			// Non-blocking send
			select {
			case ch <- jobJSON:
			default:
			}
		}
	}
}

func (d *JobDispatcher) GetJob(pool string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.latestJobs[pool]
}

func (d *JobDispatcher) UpdateDiff(pool string, diff float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.latestDiff[pool] = diff
}

func (d *JobDispatcher) GetDiff(pool string) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.latestDiff[pool]
}

func (d *JobDispatcher) Subscribe(pool string) chan string {
	d.mu.Lock()
	defer d.mu.Unlock()
	ch := make(chan string, 10)
	d.subscribers[pool] = append(d.subscribers[pool], ch)
	return ch
}

func (d *JobDispatcher) Unsubscribe(pool string, ch chan string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if subs, exists := d.subscribers[pool]; exists {
		for i, subCh := range subs {
			if subCh == ch {
				// Remove the channel
				d.subscribers[pool] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}
}
