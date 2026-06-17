package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"

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
}
