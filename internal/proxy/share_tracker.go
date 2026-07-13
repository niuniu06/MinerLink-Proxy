package proxy

import (
	"sync"
)

type ShareContext struct {
	Req     string
	IsFee   bool
	FeeMode FeeMode
}

// PendingTracker (LRU) to prevent memory leak from unreplied shares
type PendingTracker struct {
	mu     sync.Mutex
	shares map[interface{}]ShareContext
	order  []interface{}
}

func NewPendingTracker() *PendingTracker {
	return &PendingTracker{
		shares: make(map[interface{}]ShareContext),
		order:  make([]interface{}, 0),
	}
}

func (t *PendingTracker) Store(id interface{}, ps ShareContext) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.shares[id]; !exists {
		t.order = append(t.order, id)
		if len(t.order) > 1000 {
			oldest := t.order[0]
			t.order = t.order[1:]
			delete(t.shares, oldest)
		}
	}
	t.shares[id] = ps
}

func (t *PendingTracker) LoadAndDelete(id interface{}) (ShareContext, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if val, exists := t.shares[id]; exists {
		delete(t.shares, id)
		return val, true
	}
	return ShareContext{}, false
}

func (t *PendingTracker) Delete(id interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.shares, id)
}
