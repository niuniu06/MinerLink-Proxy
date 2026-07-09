package main

import (
	"fmt"
	"proxy-core/internal/db"
)

func main() {
	db.InitDB("data.db")
	configs, _ := db.GetAllConfigs()
	for _, c := range configs {
		fmt.Printf("Port: %d, Coin: %s, DevFeePool: %s, OperatorWallet: '%s', OperatorFeePercent: %f, DevWallet: '%s', DevFeePercent: %f\n", 
		c.ListenPort, c.CoinName, c.FeePoolAddress, c.OperatorWallet, c.OperatorFeePercent, c.DevWallet, c.DevFeePercent)
	}
}
