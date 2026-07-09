package models

import "time"

// EventLog stores connect/disconnect events for miners.
type EventLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Timestamp   time.Time `gorm:"index" json:"timestamp"`
	MinerIP     string    `gorm:"index;size:50" json:"minerIp"`
	MinerWorker string    `gorm:"size:100" json:"minerWorker"`
	EventType   string    `gorm:"size:20" json:"eventType"` // "ONLINE" or "OFFLINE"
	Message     string    `gorm:"size:255" json:"message"`
}
