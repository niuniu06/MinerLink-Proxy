package main

import (
	"flag"
	"log"

	"proxy-core/internal/api"
	"proxy-core/internal/db"
	"proxy-core/internal/logger"
	"proxy-core/internal/proxy"
)

func main() {
	// Parse flags
	apiPort := flag.Int("api-port", 8080, "Port for the Web UI API")
	flag.Parse()

	// 1. Init Database
	db.InitDB("./data/proxy.db")

	// 2. Fetch global config
	globalCfg, err := db.GetGlobalConfig()
	enableLogging := true
	if err == nil && globalCfg != nil {
		enableLogging = globalCfg.EnableLogging
	}

	// 3. Init global logger based on DB toggle
	logger.InitLogger(enableLogging)

	log.Println("Starting Transparent Proxy Engine (Golang Core) ...")

	// Check global config for web port override
	finalPort := *apiPort
	if err == nil && globalCfg != nil && globalCfg.WebPort > 0 {
		finalPort = globalCfg.WebPort
	}

	// 2. Init Proxy Manager and load existing proxies
	pm := proxy.NewManager()
	pm.LoadAllAndStart()

	// 3. Start API Server
	apiServer := api.NewAPIServer(pm)
	log.Printf("API Server listening on :%d\n", finalPort)
	if err := apiServer.Start(finalPort); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}

