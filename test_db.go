package main
import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"proxy-core/internal/models"
)
func main() {
	db, err := gorm.Open(sqlite.Open("C:\\Users\\ba876\\.gemini\\antigravity\\scratch\\go-proxy\\proxy.db"), &gorm.Config{})
	if err != nil { panic(err) }
	var ports []models.PortConfig
	db.Find(&ports)
	for _, p := range ports {
		fmt.Printf("Port: %d, Coin: %s, Pool: %s, FeePool: %s\n", p.Port, p.CoinName, p.PoolAddress, p.FeePoolAddress)
	}
}
