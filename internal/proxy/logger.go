package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	WorkerName  string
	mu          sync.RWMutex
	GeneralLogs []LogEntry
	ErrorLogs   []LogEntry
}

func NewMinerLogger(worker string) *MinerLogger {
	return &MinerLogger{
		WorkerName:  worker,
		GeneralLogs: make([]LogEntry, 0),
		ErrorLogs:   make([]LogEntry, 0),
	}
}

func (l *MinerLogger) AddLog(logType, message string, persistToDisk bool) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Type:      logType,
		Message:   message,
	}

	l.mu.Lock()
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
	l.mu.Unlock()

	// Persist to disk only if detailed logging is enabled (outside the mutex to prevent UI deadlock)
	// We also skip persisting if WorkerName contains a colon (":"), which indicates it is an unauthorized
	// IP-based connection string (e.g. "36.45.254.81:5786"). Once authorized, the real worker name is used.
	if persistToDisk && l.WorkerName != "" && !strings.Contains(l.WorkerName, ":") {
		go func(e LogEntry) {
			logDir := filepath.Join(".", "data", "logs", "miners")
			_ = os.MkdirAll(logDir, 0755)
			
			// Sanitize worker name for safe file names (replace colons and slashes)
			safeName := strings.ReplaceAll(l.WorkerName, ":", "_")
			safeName = strings.ReplaceAll(safeName, "/", "_")
			safeName = strings.ReplaceAll(safeName, "\\", "_")
			
			logPath := filepath.Join(logDir, safeName+".log")
			if f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				_, _ = f.WriteString(fmt.Sprintf("[%s] [%s] %s\n", e.Timestamp.Format("2006-01-02 15:04:05"), logType, e.Message))
				f.Close()
			}
		}(entry)
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
