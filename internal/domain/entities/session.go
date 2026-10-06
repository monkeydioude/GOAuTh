package entities

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SessionRevokedLogout             string = "logout"
	SessionRevokedByUser             string = "revoked_by_user"
	SessionRevokedLogoutAll          string = "logout_all"
	SessionRevokedPasswordChanged    string = "password_changed"
	SessionRevokedPasswordReset      string = "password_reset"
	SessionRevokedAccountDeactivated string = "account_deactivated"
	SessionRevokedReuseDetected      string = "reuse_detected"
	SessionRevokedLimitExceeded      string = "limit_exceeded"
)

// Session is one login of a user. Its ID is the sid claim of the tokens issued for it.
type Session struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID            uint      `gorm:"not null;index"`
	User              *User     `gorm:"constraint:OnDelete:CASCADE;"`
	TokenHash         string    `gorm:"not null;uniqueIndex"` // SHA-256 of the current refresh token
	PreviousTokenHash *string   `gorm:"index"`                // replaced by the last rotation
	RotatedAt         *time.Time
	UserAgent         string
	LoginIP           string
	LastIP            string
	CreatedAt         time.Time
	LastConnection    time.Time `gorm:"not null"`
	ExpiresAt         time.Time `gorm:"not null"`
	RevokedReason     *string
	DeletedAt         gorm.DeletedAt `gorm:"index"` // set = revoked
}

func (Session) TableName() string {
	return "sessions"
}

// BeforeCreate is a GORM hook impl
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.UserID == 0 || s.TokenHash == "" {
		slog.Error("user_id or token_hash cannot be empty")
		return errors.New("user_id or token_hash cannot be empty")
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
