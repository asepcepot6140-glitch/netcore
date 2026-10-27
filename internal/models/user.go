package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Email     string    `gorm:"size:150;uniqueIndex;not null" json:"email"`
	Phone     string    `gorm:"size:20;uniqueIndex;not null" json:"phone"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
