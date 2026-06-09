package proxy

import (
	"sync"
	"time"
)

const (
	LogTypeGeneral = "general"
	LogTypeError   = "error"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
}

type MinerLogger struct {
	mu          sync.RWMutex
	GeneralLogs []LogEntry
	ErrorLogs   []LogEntry
}

func NewMinerLogger() *MinerLogger {
	return &MinerLogger{
		GeneralLogs: make([]LogEntry, 0),
		ErrorLogs:   make([]LogEntry, 0),
	}
}

func (l *MinerLogger) AddLog(logType, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := LogEntry{
		Timestamp: time.Now(),
		Type:      logType,
		Message:   message,
	}
	if logType == LogTypeError {
		l.ErrorLogs = append(l.ErrorLogs, entry)
		if len(l.ErrorLogs) > 50 {
			l.ErrorLogs = l.ErrorLogs[len(l.ErrorLogs)-50:]
		}
	} else {
		l.GeneralLogs = append(l.GeneralLogs, entry)
		if len(l.GeneralLogs) > 50 {
			l.GeneralLogs = l.GeneralLogs[len(l.GeneralLogs)-50:]
		}
	}
}

func (l *MinerLogger) Prune() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	
	// Prune general logs older than 5 minutes (max 50 items)
	genFiltered := make([]LogEntry, 0)
	for _, entry := range l.GeneralLogs {
		if now.Sub(entry.Timestamp) <= 5*time.Minute {
			genFiltered = append(genFiltered, entry)
		}
	}
	if len(genFiltered) > 50 {
		genFiltered = genFiltered[len(genFiltered)-50:]
	}
	l.GeneralLogs = genFiltered

	// Prune error logs older than 24 hours (max 50 items)
	errFiltered := make([]LogEntry, 0)
	for _, entry := range l.ErrorLogs {
		if now.Sub(entry.Timestamp) <= 24*time.Hour {
			errFiltered = append(errFiltered, entry)
		}
	}
	if len(errFiltered) > 50 {
		errFiltered = errFiltered[len(errFiltered)-50:]
	}
	l.ErrorLogs = errFiltered
}
