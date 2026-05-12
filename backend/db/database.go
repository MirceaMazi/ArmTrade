package db

import (
	"log"

	"armtrade-backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDatabase() {
	var err error
	DB, err = gorm.Open(sqlite.Open("armtrade.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate models
	if err := DB.AutoMigrate(&models.User{}, &models.WatchlistItem{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database initialized successfully (armtrade.db)")
}
