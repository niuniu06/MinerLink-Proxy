package main

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

func testMethod(name, authParam, suggest string) {
	fmt.Printf("\n--- Testing %s ---\n", name)
	conn, err := net.DialTimeout("tcp", "btc.f2pool.com:3333", 5*time.Second)
	if err != nil {
		fmt.Println("Dial error:", err)
		return
	}
	defer conn.Close()

	conn.Write([]byte("{\"id\": 1, \"method\": \"mining.subscribe\", \"params\": [\"cgminer/4.11.1\"]}\n"))
	
	if suggest != "" {
		conn.Write([]byte(suggest + "\n"))
	}
	
	conn.Write([]byte("{\"id\": 2, \"method\": \"mining.authorize\", \"params\": [\"boniu.bate001\", \"" + authParam + "\"]}\n"))

	scanner := bufio.NewScanner(conn)
	timeout := time.Now().Add(5 * time.Second)
	for time.Now().Before(timeout) {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if scanner.Scan() {
			fmt.Println("RX:", scanner.Text())
		} else {
			break
		}
	}
}

func main() {
	testMethod("Standard (x)", "x", "")
	testMethod("Password (d=2097152)", "d=2097152", "")
	testMethod("Password (123,d=2097152)", "123,d=2097152", "")
	testMethod("Suggest Difficulty", "x", "{\"id\": 3, \"method\": \"mining.suggest_difficulty\", \"params\": [2097152]}")
}
