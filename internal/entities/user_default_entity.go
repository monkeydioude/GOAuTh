package entities

import (
	"GOAuTh/pkg/crypt"
	"GOAuTh/pkg/entities"
	"errors"
	"time"

	"gorm.io/gorm"
)

type DefaultUser struct {
	entities.Entity[UsersParams] `gorm:"-" json:"-"`
	ID                           uint           `gorm:"primaryKey;autoIncrement" json:"-"`
	Login                        string         `gorm:"unique;not null" json:"login"`
	Password                     string         `gorm:"not null" json:"password,omitempty"`
	CreatedAt                    time.Time      `json:"-"`
	UpdatedAt                    time.Time      `json:"-"`
	LastLoggedAt                 *time.Time     `json:"-"`
	RevokedAt                    *time.Time     `json:"-"`
	DeletedAt                    gorm.DeletedAt `gorm:"index" json:"-"`
	Parameters                   *UsersParams   `gorm:"-" json:"-"`
}

// SetParameters impl for the GOAuTh/pkg/entities.Entity interface
func (u *DefaultUser) SetParameters(p entities.Parameters) error {
	t, ok := p.(*UsersParams)
	if !ok {
		return errors.New("could not cast type in SetParameters")
	}
	u.Parameters = t
	return nil
}

// BeforeCreate is a GORM hook impl
func (u *DefaultUser) BeforeCreate(tx *gorm.DB) error {
	if u.Login == "" || u.Password == "" {
		return errors.New("login or password cannot be empty")
	}
	if u.Password != "" {
		u.Password = crypt.HashPassword(
			u.Password,
			u.Parameters.GetArgon2Params(),
			u.Parameters.GetPasswordSalt(),
		)
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	return nil
}

// BeforeUpdate is a GORM hook impl
func (u *DefaultUser) BeforeUpdate(tx *gorm.DB) error {
	if u.Password != "" {
		u.Password = crypt.HashPassword(
			u.Password,
			u.Parameters.GetArgon2Params(),
			u.Parameters.GetPasswordSalt(),
		)
	}
	u.UpdatedAt = time.Now()
	return nil
}

func (DefaultUser) TableName() string {
	return "users"
}

// AssertAuth asserts a user can pass authentification.
func (u *DefaultUser) AssertAuth(db *gorm.DB) error {
	passwd := ""
	if u.Password != "" {
		passwd = crypt.HashPassword(
			u.Password,
			u.Parameters.GetArgon2Params(),
			u.Parameters.GetPasswordSalt(),
		)
	}
	return db.First(u, "login = ? AND password = ?", u.Login, passwd).Error
}

func (u DefaultUser) IntoClaims() crypt.JWTDefaultClaims {
	return crypt.JWTDefaultClaims{
		Name: u.Login,
	}
}

func (u DefaultUser) IsRevoked(timeRef time.Time) bool {
	return u.RevokedAt != nil && u.RevokedAt.Before(timeRef)
}

func NewDefaultUser() *DefaultUser {
	return &DefaultUser{}
}
