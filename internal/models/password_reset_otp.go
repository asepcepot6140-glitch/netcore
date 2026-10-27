package models

import (
	"time"
)

type PasswordResetOTP struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;index" json:"user_id"`
	OTPHash   string     `gorm:"size:255;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	Attempts  int        `gorm:"not null;default:0" json:"-"`
	UsedAt    *time.Time `json:"-"`
	CreatedAt time.Time  `gorm:"not null" json:"created_at"`
}

func (PasswordResetOTP) TableName() string {
	return "password_reset_otps"
}
