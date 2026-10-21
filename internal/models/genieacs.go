package models

import "time"

type GenieACS struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Host      string    `gorm:"size:255;not null" json:"host"`
	Port      int       `gorm:"not null;default:7557" json:"port"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (GenieACS) TableName() string {
	return "genieacs"
}
