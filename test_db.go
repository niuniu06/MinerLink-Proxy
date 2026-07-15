package main

import (
	"fmt"
	"proxy-core/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("data/proxy.db"), &gorm.Config{})
	if err != nil {
		fmt.Println(err)
		return
	}
	var configs []models.ProxyConfig
	db.Find(&configs)
	for _, c := range configs {
		fmt.Printf("Port: %d, EnableAsic: %v, DevFee: %v, OpFee: %v\n", c.ListenPort, c.EnableAsic, c.DevFeePercent, c.OperatorFeePercent)
	}
}
