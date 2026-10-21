package models

import "time"

type Mikrotik struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"not null;index" json:"user_id"`
	Name              string    `gorm:"size:100;not null" json:"name"`
	Host              string    `gorm:"size:255;not null" json:"host"`
	Username          string    `gorm:"size:100;not null" json:"username"`
	PasswordEncrypted string    `gorm:"type:text;not null" json:"-"`
	Port              int       `gorm:"not null;default:8728" json:"port"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Mikrotik) TableName() string {
	return "mikrotiks"
}
