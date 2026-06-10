package models

import (
	"time"

	"gorm.io/gorm"
)

// ProxyConfig stores the configuration for a single mining proxy cluster (port)
type ProxyConfig struct {
	ID                 uint           `gorm:"primarykey" json:"id"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
	
	CoinName           string  `json:"coinName"`
	ListenPort         int     `gorm:"uniqueIndex" json:"listenPort"`
	PoolAddress        string  `json:"poolAddress"`
	FeePoolAddress     string  `json:"feePoolAddress"`
	
	OperatorWallet     string  `json:"operatorWallet"`
	OperatorWorker     string  `json:"operatorWorker"`
	OperatorFeePercent float64 `json:"operatorFeePercent"`
	
	DevWallet          string  `json:"devWallet"`
	DevWorker          string  `json:"devWorker"`
	DevFeePercent      float64 `json:"devFeePercent"`
	
	HashrateMultiplier float64 `json:"hashrateMultiplier"`
	HashrateUnit       string  `json:"hashrateUnit"`
	
	// Advanced settings
	EnableSmoothFee    bool    `json:"enableSmoothFee"`
	EnableAsic         bool    `json:"enableAsic"`
	EnableAntiBan      bool    `json:"enableAntiBan"`
	EnableDetailedLog  bool    `json:"enableDetailedLog"`
	IsViaBtcOptimize   bool    `json:"isViaBtcOptimize"`
	MainFixedDifficulty string  `json:"mainFixedDifficulty"`
	FeeFixedDifficulty  string  `json:"feeFixedDifficulty"`
	WebhookUrl         string  `json:"webhookUrl"`
	AutoRestart        bool    `json:"autoRestart"`
	FeeCycleMinutes    int     `json:"feeCycleMinutes"`

	EnableStaleDrop      bool    `json:"enableStaleDrop"`
	EnableEthTargetRewrite bool  `json:"enableEthTargetRewrite"`
	EnableTcpNoDelay     bool    `json:"enableTcpNoDelay"`

	// Auto-Vardiff settings
	EnableVardiff      bool    `json:"enableVardiff"`
	TargetShareRate    int     `json:"targetShareRate"`
}
