package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Realm struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name         string         `gorm:"unique;not null" json:"name"`
	Description  string         `gorm:"not null;default:''" json:"description,omitempty"`
	AllowNewUser bool           `gorm:";not null" json:"allow_new_user"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Realm) TableName() string {
	return "realms"
}

func (r *Realm) BeforeCreate(tx *gorm.DB) error {
	r.ID = uuid.New()
	r.CreatedAt = time.Now()
	return nil
}

func (r *Realm) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now()
	return nil
}
