package entities

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type DefaultUser struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"-"`
	Login        string         `gorm:"unique;not null" json:"login"`
	Password     string         `gorm:"not null" json:"password,omitempty"`
	CreatedAt    time.Time      `json:"-"`
	UpdatedAt    time.Time      `json:"-"`
	LastLoggedAt *time.Time     `json:"-"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *DefaultUser) BeforeCreate(tx *gorm.DB) error {
	if u.Login == "" || u.Password == "" {
		return errors.New("login or password cannot be empty")
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	return nil
}

func (u *DefaultUser) BeforeUpdate(tx *gorm.DB) error {
	if u.Login == "" || u.Password == "" {
		return errors.New("login or password cannot be empty")
	}
	u.UpdatedAt = time.Now()
	return nil
}
