package entities

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserActionType string

const (
	UserActionTypePassword UserActionType = "reset-password"
	UserActionTypeActivate UserActionType = "activate-account"
)

func (uat UserActionType) String() string {
	return string(uat)
}

func UserActionTypeFromString(s string) (UserActionType, error) {
	switch s {
	case UserActionTypePassword.String():
		return UserActionTypePassword, nil
	case UserActionTypeActivate.String():
		return UserActionTypeActivate, nil
	}

	return "", errors.New("invalid UserActionType")
}

type UserAction struct {
	ID          uint           `gorm:"primaryKey;autoIncrement"`
	UserID      uint           `gorm:"index"`
	RealmID     uuid.UUID      `gorm:"index"`
	Action      UserActionType `gorm:"index"`
	Data        string         `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ValidatedAt *time.Time
}

func (UserAction) TableName() string {
	return "user_actions"
}

// BeforeCreate is a GORM hook impl
func (ua *UserAction) BeforeCreate(tx *gorm.DB) error {
	if tx == nil {
		slog.Error("nil *gorm.DB")
		return errors.New("nil *gorm.DB")
	}
	if ua.UserID == 0 || ua.RealmID.String() == "" || ua.Data == "" {
		slog.Error("user_id, realm_id or data cannot be empty")
		return errors.New("user_id, realm_id or data cannot be empty")
	}
	ua.CreatedAt = time.Now()
	ua.UpdatedAt = ua.CreatedAt
	return nil
}

// BeforeUpdate is a GORM hook impl
func (ua *UserAction) BeforeUpdate(tx *gorm.DB) error {
	ua.UpdatedAt = time.Now()
	return nil
}
