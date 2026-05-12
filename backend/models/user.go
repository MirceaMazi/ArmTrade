package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:100;not null" json:"username"`
	PasswordHash string         `gorm:"not null" json:"-"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"-"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
