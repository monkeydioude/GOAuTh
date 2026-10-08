package entities

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AccessKeyRevokedManual         string = "manual"
	AccessKeyRevokedAccountDeleted string = "account_deleted"
	AccessKeyNameMaxLength                = 100
)

// AccessKey is a long-lived secret of an account, shown once at creation and
// kept as a hash. ID is the key_id a consumer names it by. RevokedAt is the
// soft-delete mark, so queries see live keys only unless Unscoped.
type AccessKey struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	AccountID     uint      `gorm:"not null;index"`
	Account       *User     `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE;"`
	Name          string    `gorm:"not null"`
	KeyHash       string    `gorm:"not null;uniqueIndex"` // SHA-256 of the key
	Prefix        string    `gorm:"not null"`             // the key's first characters after its prefix, to tell keys apart
	CreatedAt     time.Time `gorm:"not null"`
	CreatedBy     string    `gorm:"not null"` // who asked for it, as the consumer names them
	ExpiresAt     *time.Time
	LastUsedAt    *time.Time
	RevokedBy     *string
	RevokedReason *string
	RevokedAt     gorm.DeletedAt `gorm:"index"`
}

func (AccessKey) TableName() string {
	return "access_keys"
}

// BeforeCreate is a GORM hook impl
func (k *AccessKey) BeforeCreate(tx *gorm.DB) error {
	if k.AccountID == 0 || k.KeyHash == "" || k.Name == "" || k.CreatedBy == "" {
		return errors.New("account_id, key_hash, name or created_by cannot be empty")
	}
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}

// IsExpired tells whether the key's expiry, when it has one, has passed.
func (k AccessKey) IsExpired(now time.Time) bool {
	return k.ExpiresAt != nil && !k.ExpiresAt.After(now)
}
