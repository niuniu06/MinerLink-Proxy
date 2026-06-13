//go:build windows

package sysinfo

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

type FILETIME struct {
	DwLowDateTime  uint32
	DwHighDateTime uint32
}

func (ft *FILETIME) ToNano() int64 {
	return (int64(ft.DwHighDateTime) << 32) + int64(ft.DwLowDateTime)
}

type MEMORYSTATUSEX struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

var lastIdleTime FILETIME
var lastKernelTime FILETIME
var lastUserTime FILETIME
var initialized bool

func getSystemCPUPercent() float64 {
	var idleTime, kernelTime, userTime FILETIME
	ret, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if ret == 0 {
		return 0.0
	}

	if !initialized {
		lastIdleTime = idleTime
		lastKernelTime = kernelTime
		lastUserTime = userTime
		initialized = true
		time.Sleep(100 * time.Millisecond) // Give a brief sleep for initial difference
		ret, _, _ = procGetSystemTimes.Call(
			uintptr(unsafe.Pointer(&idleTime)),
			uintptr(unsafe.Pointer(&kernelTime)),
			uintptr(unsafe.Pointer(&userTime)),
		)
		if ret == 0 {
			return 0.0
		}
	}

	idleDiff := idleTime.ToNano() - lastIdleTime.ToNano()
	kernelDiff := kernelTime.ToNano() - lastKernelTime.ToNano()
	userDiff := userTime.ToNano() - lastUserTime.ToNano()

	lastIdleTime = idleTime
	lastKernelTime = kernelTime
	lastUserTime = userTime

	totalDiff := kernelDiff + userDiff
	if totalDiff <= 0 {
		return 0.0
	}

	activeDiff := totalDiff - idleDiff
	if activeDiff < 0 {
		activeDiff = 0
	}
	return float64(activeDiff) * 100.0 / float64(totalDiff)
}

func getSystemMemoryPercent() float64 {
	var memInfo MEMORYSTATUSEX
	memInfo.dwLength = uint32(unsafe.Sizeof(memInfo))
	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memInfo)))
	if ret == 0 {
		return 0.0
	}
	return float64(memInfo.dwMemoryLoad)
}
