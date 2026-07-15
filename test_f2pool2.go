package main

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "btc-asia.f2pool.com:1315")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	fmt.Fprintf(conn, "{\"id\": 1, \"method\": \"mining.subscribe\", \"params\": [\"MinerLink-Proxy/2.2.117-beta\"]}\n")
	
	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		if len(scanner.Text()) > 0 {
            fmt.Fprintf(conn, "{\"params\": [16384], \"id\": 2, \"method\": \"mining.suggest_difficulty\"}\n")
            fmt.Fprintf(conn, "{\"params\": [\"linkpro168.ai\", \"password\"], \"id\": 3, \"method\": \"mining.authorize\"}\n")
            break
		}
	}
    for scanner.Scan() {
        fmt.Println(scanner.Text())
    }
}
