package proxy

import "sync"

// ScannerBufferPool is a global pool for bufio.Scanner buffers.
// It vastly reduces garbage collection overhead and memory fragmentation
// by reusing 64KB byte slices across tens of thousands of miner connections.
var ScannerBufferPool = sync.Pool{
	New: func() interface{} {
		// bufio.Scanner uses this slice as the initial buffer.
		// 64KB is generally large enough to hold any single Stratum JSON-RPC line.
		return make([]byte, 0, 64*1024)
	},
}
