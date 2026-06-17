package db

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"proxy-core/internal/models"
)

var DB *gorm.DB

// InitDB initializes the SQLite database
func InitDB(dbPath string) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migrate the schema
	err = DB.AutoMigrate(&models.ProxyConfig{}, &models.GlobalConfig{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database initialized successfully at", dbPath)
}

// GetAllConfigs retrieves all proxy configurations
func GetAllConfigs() ([]models.ProxyConfig, error) {
	var configs []models.ProxyConfig
	result := DB.Find(&configs)
	return configs, result.Error
}

// SaveConfig creates or updates a proxy configuration based on listen port
func SaveConfig(config *models.ProxyConfig) error {
	var existing models.ProxyConfig
	result := DB.Unscoped().Where("listen_port = ?", config.ListenPort).First(&existing)
	if result.Error == nil {
		// Update
		config.ID = existing.ID
		if existing.DeletedAt.Valid {
			// Restore soft-deleted record
			DB.Unscoped().Model(&existing).Update("deleted_at", nil)
		}
		return DB.Save(config).Error
	}
	// Create
	return DB.Create(config).Error
}

// DeleteConfig deletes a proxy configuration by port
func DeleteConfig(port int) error {
	return DB.Where("listen_port = ?", port).Delete(&models.ProxyConfig{}).Error
}

// GetGlobalConfig retrieves the global configuration (creates default if not exists)
func GetGlobalConfig() (*models.GlobalConfig, error) {
	var cfg models.GlobalConfig
	result := DB.First(&cfg)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Initialize default
			cfg = models.GlobalConfig{WebPort: 0, EnableLogging: true}
			DB.Create(&cfg)
			return &cfg, nil
		}
		return nil, result.Error
	}
	return &cfg, nil
}

// SaveGlobalConfig saves the global configuration
func SaveGlobalConfig(cfg *models.GlobalConfig) error {
	var existing models.GlobalConfig
	result := DB.First(&existing)
	if result.Error == nil {
		cfg.ID = existing.ID
		return DB.Save(cfg).Error
	}
	return DB.Create(cfg).Error
}

// AddSafeMiner adds a miner to the SafeMiners list for a specific proxy
func AddSafeMiner(port int, minerIdent string) error {
	var config models.ProxyConfig
	result := DB.Unscoped().Where("listen_port = ?", port).First(&config)
	if result.Error != nil {
		return result.Error
	}

	if !strings.Contains(config.SafeMiners, minerIdent) {
		if config.SafeMiners == "" {
			config.SafeMiners = minerIdent
		} else {
			config.SafeMiners += "," + minerIdent
		}
		return DB.Save(&config).Error
	}
	return nil
}
