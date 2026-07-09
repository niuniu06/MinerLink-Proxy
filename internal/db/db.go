package db

import (
	"log"
	"os"
	"path/filepath"
	"time"

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
	err = DB.AutoMigrate(&models.ProxyConfig{}, &models.GlobalConfig{}, &models.HashrateHistory{}, &models.EventLog{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	

	// Hotfix for v2.0.89 to v2.0.90 migration bug where all existing configs got Enabled=false
	// We only run this ONCE. We track it using MigratedEnabled in GlobalConfig.
	var globalCfg models.GlobalConfig
	if err := DB.First(&globalCfg).Error; err == nil {
		if !globalCfg.MigratedEnabled {
			// Check if we need to apply the hotfix
			var totalConfigs int64
			DB.Model(&models.ProxyConfig{}).Count(&totalConfigs)
			if totalConfigs > 0 {
				var enabledConfigs int64
				DB.Model(&models.ProxyConfig{}).Where("enabled = ?", true).Count(&enabledConfigs)
				if enabledConfigs == 0 {
					DB.Model(&models.ProxyConfig{}).Where("enabled = ?", false).Update("enabled", true)
					log.Println("Applied migration hotfix: Enabled all proxy configs.")
				}
			}
			// Mark as migrated
			globalCfg.MigratedEnabled = true
			DB.Save(&globalCfg)
		}
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
			cfg = models.GlobalConfig{WebPort: 0, EnableLogging: true, MigratedEnabled: true}
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




// startCleanupTask periodically cleans up history data older than 48 hours
func startCleanupTask() {
	for {
		time.Sleep(1 * time.Hour)
		if DB != nil {
			cutoff := time.Now().Add(-48 * time.Hour)
			DB.Where("timestamp < ?", cutoff).Delete(&models.HashrateHistory{})
			DB.Where("timestamp < ?", cutoff).Delete(&models.EventLog{})
		}
	}
}
