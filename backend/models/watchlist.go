package models

import (
	"time"

	"gorm.io/gorm"
)

type WatchlistItem struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"userId"`
	Ticker    string         `gorm:"size:20;not null" json:"ticker"`
	AddedAt   time.Time      `json:"addedAt"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
}
