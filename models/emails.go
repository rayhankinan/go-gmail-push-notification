package models

import "gorm.io/gorm"

type Email struct {
	gorm.Model
	Email           string `gorm:"unique"`
	Expiration      int64
	LatestHistoryID uint64
}
