package main
import (
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"proxy-core/internal/models"
)
func main() {
	db, _ := gorm.Open(sqlite.Open("data/proxy.db"), &gorm.Config{})
	var configs []models.ProxyConfig
	db.Find(&configs)
	for _, c := range configs {
		fmt.Printf("Port: %d, Coin: %s, DevFee: %v, DevWallet: %v, OpFee: %v\n", c.ListenPort, c.CoinName, c.DevFeePercent, c.DevWallet, c.OperatorFeePercent)
	}
}
