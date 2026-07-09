package main

import (
	"fmt"
	"proxy-core/internal/db"
)

func main() {
	db.InitDB("data.db")
	configs, _ := db.GetAllConfigs()
	for _, c := range configs {
		fmt.Printf("Port: %d, Coin: %s, DevPercent: %f, OpPercent: %f\n", c.ListenPort, c.CoinName, c.DevFeePercent, c.OperatorFeePercent)
	}
}
