package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Global Log file path
var LogFilePath = filepath.Join("data", "logs", "proxy.log")

// InitLogger sets up global logging to both os.Stdout and a rotating file if enabled
func InitLogger(enableFile bool) {
	if enableFile {
		// Ensure the log directory exists
		err := os.MkdirAll(filepath.Dir(LogFilePath), 0755)
		if err != nil {
			log.Printf("Failed to create log directory: %v", err)
		}

		fileLogger := &lumberjack.Logger{
			Filename:   LogFilePath,
			MaxSize:    50, // megabytes
			MaxBackups: 3,
			MaxAge:     28, // days
			Compress:   true, // disabled by default
		}

		multiWriter := io.MultiWriter(os.Stdout, fileLogger)
		log.SetOutput(multiWriter)
	} else {
		log.SetOutput(os.Stdout)
	}
	
	// Add timestamps and file lines if desired
	log.SetFlags(log.Ldate | log.Ltime)

	StartMinerLogCleanupRoutine()
}

// StartMinerLogCleanupRoutine runs in the background and ensures the miners log directory doesn't exceed 1GB
func StartMinerLogCleanupRoutine() {
	go func() {
		for {
			time.Sleep(10 * time.Minute) // Check every 10 minutes
			logDir := filepath.Join("data", "logs", "miners")
			entries, err := os.ReadDir(logDir)
			if err != nil {
				continue
			}

			var totalSize int64
			type fileInfo struct {
				path    string
				modTime int64
				size    int64
			}
			var files []fileInfo

			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				totalSize += info.Size()
				files = append(files, fileInfo{
					path:    filepath.Join(logDir, e.Name()),
					modTime: info.ModTime().UnixNano(),
					size:    info.Size(),
				})
			}

			const maxSizeBytes = 1024 * 1024 * 1024 // 1 GB
			if totalSize > maxSizeBytes {
				// Simple bubble sort or any sort to order by oldest first without importing sort package,
				// actually let's just import sort package at the top. Wait, we can just write a simple bubble sort.
				for i := 0; i < len(files)-1; i++ {
					for j := 0; j < len(files)-i-1; j++ {
						if files[j].modTime > files[j+1].modTime {
							files[j], files[j+1] = files[j+1], files[j]
						}
					}
				}

				// Target size is 800MB after cleanup to leave breathing room
				targetSize := int64(800 * 1024 * 1024)
				for _, f := range files {
					if totalSize <= targetSize {
						break
					}
					if err := os.Remove(f.path); err == nil {
						totalSize -= f.size
					}
				}
			}
		}
	}()
}
