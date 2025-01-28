package entities

import (
	"time"

	"gorm.io/gorm"
)

type Realm struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string         `gorm:"unique;not null" json:"name"`
	AllowNewUser bool           `gorm:";not null" json:"allow_new_user"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
