//go:build !windows

package sysinfo

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

var lastCPUStats []uint64
var lastSampleTime time.Time

func getSystemCPUPercent() float64 {
	stats, err := readProcStat()
	if err != nil || len(stats) < 4 {
		return 0.0
	}

	now := time.Now()
	if lastCPUStats == nil {
		lastCPUStats = stats
		lastSampleTime = now
		time.Sleep(100 * time.Millisecond) // Give a brief sleep for initial difference
		stats, err = readProcStat()
		if err != nil || len(stats) < 4 {
			return 0.0
		}
	}

	idle := stats[3] + stats[4] // idle + iowait
	total := uint64(0)
	for _, val := range stats {
		total += val
	}

	lastIdle := lastCPUStats[3] + lastCPUStats[4]
	lastTotal := uint64(0)
	for _, val := range lastCPUStats {
		lastTotal += val
	}

	lastCPUStats = stats
	lastSampleTime = now

	totalDiff := total - lastTotal
	idleDiff := idle - lastIdle

	if totalDiff == 0 {
		return 0.0
	}

	activeDiff := totalDiff - idleDiff
	if activeDiff > totalDiff {
		activeDiff = totalDiff
	}
	return float64(activeDiff) * 100.0 / float64(totalDiff)
}

func readProcStat() ([]uint64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] == "cpu" {
			var stats []uint64
			for i := 1; i < len(fields); i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					continue
				}
				stats = append(stats, val)
			}
			return stats, nil
		}
	}
	return nil, scanner.Err()
}

func getSystemMemoryStats() (float64, float64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0.0, 0.0
	}
	defer file.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		if key == "MemTotal" {
			memTotal = val
		} else if key == "MemAvailable" {
			memAvailable = val
		}
	}

	if memTotal == 0 {
		return 0.0, 0.0
	}

	if memAvailable == 0 {
		// Fallback for older Linux kernels: memFree + buffers + cached
		var memFree, buffers, cached uint64
		file.Seek(0, 0)
		scanner = bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			key := strings.TrimSuffix(fields[0], ":")
			val, _ := strconv.ParseUint(fields[1], 10, 64)
			switch key {
			case "MemFree":
				memFree = val
			case "Buffers":
				buffers = val
			case "Cached":
				cached = val
			}
		}
		memAvailable = memFree + buffers + cached
	}

	if memAvailable > memTotal {
		memAvailable = memTotal
	}
	used := memTotal - memAvailable
	return float64(used) * 100.0 / float64(memTotal), float64(memTotal) / 1024.0
}
