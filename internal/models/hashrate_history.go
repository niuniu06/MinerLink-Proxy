package models

import "time"

// HashrateHistory stores historical hashrate data points for the 24-hour curve.
type HashrateHistory struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Timestamp    time.Time `gorm:"index" json:"timestamp"`
	MinerIP      string    `gorm:"index;size:50" json:"minerIp"` // empty means global hashrate
	Port         int       `gorm:"index" json:"port"`            // port number for port-level hashrate
	CoinName     string    `gorm:"size:20" json:"coinName"`
	MainHashrate float64   `json:"mainHashrate"`
	FeeHashrate  float64   `json:"feeHashrate"`
}
