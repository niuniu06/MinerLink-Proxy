package models

import (
	"gorm.io/gorm"
)

// GlobalConfig represents the system-wide settings
type GlobalConfig struct {
	gorm.Model
	WebPort       int  `json:"webPort"`
	EnableLogging bool `json:"enableLogging"`
}
