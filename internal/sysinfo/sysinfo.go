package sysinfo

import (
	"runtime"
	"time"
)

const ProxyVersion = "v2.2.58"

type SystemStatus struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	ProcMemMB     float64 `json:"procMemMB"`
	TotalMemMB    float64 `json:"totalMemMB"`
	UptimeSeconds int64   `json:"uptimeSeconds"`
	Version       string  `json:"version"`
}

var startTime = time.Now()

func GetSystemStatus() SystemStatus {
	var status SystemStatus
	status.UptimeSeconds = int64(time.Since(startTime).Seconds())
	status.CPUPercent = getSystemCPUPercent()
	status.MemoryPercent, status.TotalMemMB = getSystemMemoryStats()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	status.ProcMemMB = float64(m.Sys) / 1024.0 / 1024.0

	status.Version = ProxyVersion

	// Format to 1 decimal place
	if status.CPUPercent < 0 {
		status.CPUPercent = 0.0
	} else if status.CPUPercent > 100 {
		status.CPUPercent = 100.0
	}
	status.CPUPercent = float64(int(status.CPUPercent*10)) / 10.0

	if status.MemoryPercent < 0 {
		status.MemoryPercent = 0.0
	} else if status.MemoryPercent > 100 {
		status.MemoryPercent = 100.0
	}
	status.MemoryPercent = float64(int(status.MemoryPercent*10)) / 10.0
	status.ProcMemMB = float64(int(status.ProcMemMB*10)) / 10.0
	status.TotalMemMB = float64(int(status.TotalMemMB*10)) / 10.0

	return status
}
