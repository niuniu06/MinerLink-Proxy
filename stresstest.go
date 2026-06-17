package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	target := flag.String("target", "127.0.0.1:1800", "Proxy listening port")
	conns := flag.Int("c", 1000, "Number of concurrent connections")
	flag.Parse()

	fmt.Printf("🚀 Starting High-Concurrency Stress Test against %s with %d connections...\n", *target, *conns)

	var wg sync.WaitGroup
	var activeConns int32

	for i := 0; i < *conns; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", *target, 5*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()

			atomic.AddInt32(&activeConns, 1)
			defer atomic.AddInt32(&activeConns, -1)

			loginMsg := fmt.Sprintf(`{"id": 1, "method": "mining.subscribe", "params": ["stressminer/1.0"]}`+"\n")
			_, _ = conn.Write([]byte(loginMsg))
			authMsg := fmt.Sprintf(`{"id": 2, "method": "mining.authorize", "params": ["worker-%d", "x"]}`+"\n", id)
			_, _ = conn.Write([]byte(authMsg))

			for {
				time.Sleep(1 * time.Second)
				shareMsg := fmt.Sprintf(`{"id": 4, "method": "mining.submit", "params": ["worker-%d", "job1", "00000000", "00000000", "00000000"]}`+"\n", id)
				_, err := conn.Write([]byte(shareMsg))
				if err != nil {
					return
				}
			}
		}(i)
		
		time.Sleep(5 * time.Millisecond) // Avoid instant socket exhaustion
	}

	fmt.Printf("✅ All %d connections dispatched, holding for 30 seconds...\n", *conns)
	
	for i := 0; i < 30; i++ {
		fmt.Printf("-> Active Connections: %d\n", atomic.LoadInt32(&activeConns))
		time.Sleep(1 * time.Second)
	}
	
	fmt.Println("🛑 Stress test finished.")
}
