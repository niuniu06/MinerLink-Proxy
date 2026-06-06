package main

import (
	"flag"
	"fmt"
	"log"

	"proxy-core/internal/api"
	"proxy-core/internal/db"
	"proxy-core/internal/proxy"
)

func main() {
	// Parse flags
	apiPort := flag.Int("api-port", 8080, "Port for the Web UI API")
	flag.Parse()

	log.Println("Starting Transparent Proxy Engine (Golang Core) ...")

	// 1. Init Database
	db.InitDB(./data/proxy.db)

	// 2. Init Proxy Manager and load existing proxies
	pm := proxy.NewManager()
	pm.LoadAllAndStart()

	// 3. Start API Server
	apiServer := api.NewAPIServer(pm)
	log.Printf("API Server listening on :%d\n", *apiPort)
	if err := apiServer.Start(*apiPort); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}

