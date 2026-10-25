package entities

import (
	"errors"
	"github.com/monkeydioude/goauth/v2/internal/config/logs"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	UserActionTypePassword string = "reset-password"
)

type UserAction struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	UserID      uint      `gorm:"index"`
	RealmID     uuid.UUID `gorm:"index"`
	Action      string    `gorm:"index"`
	Data        string    `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ValidatedAt *time.Time
	// ExpiresAt: from then on, the action can't be validated; nil: never expires
	ExpiresAt *time.Time
	// Attempts are the wrong codes tried with a login and the action's type
	Attempts int `gorm:"not null;default:0"`
	// MaxAttempts: once Attempts reach it, the action can't be validated; 0: no limit
	MaxAttempts int `gorm:"not null;default:0"`
}

// Expired tells if the action can no longer be validated at now.
func (ua UserAction) Expired(now time.Time) bool {
	return ua.ExpiresAt != nil && !now.Before(*ua.ExpiresAt)
}

// TriesUsedUp tells if the wrong codes tried reached the action's limit.
func (ua UserAction) TriesUsedUp() bool {
	return ua.MaxAttempts > 0 && ua.Attempts >= ua.MaxAttempts
}

func (UserAction) TableName() string {
	return "user_actions"
}

// BeforeCreate is a GORM hook impl
func (ua *UserAction) BeforeCreate(tx *gorm.DB) error {
	if tx == nil {
		slog.ErrorContext(logs.DBContext(tx), "nil *gorm.DB")
		return errors.New("nil *gorm.DB")
	}
	if ua.UserID == 0 || ua.RealmID.String() == "" || ua.Data == "" {
		slog.ErrorContext(logs.DBContext(tx), "user_id, realm_id or data cannot be empty")
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
