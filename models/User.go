package models

import "gorm.io/gorm"

// User represents an application account that can authenticate with the platform.
type User struct {
	gorm.Model
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Name         string
}
