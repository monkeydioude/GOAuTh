package entities

import (
	"GOAuTh/internal/domain/models"
	"GOAuTh/pkg/crypt"
	"errors"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Login        string         `gorm:"unique;not null" json:"login"`
	Password     string         `gorm:"not null" json:"password,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	LastLoggedAt *time.Time     `json:"last_logged_at"`
	RevokedAt    *time.Time     `json:"revoked_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate is a GORM hook impl
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if tx == nil {
		return errors.New("nil *gorm.DB")
	}
	if u.Login == "" || u.Password == "" {
		return errors.New("login or password cannot be empty")
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	return nil
}

// BeforeUpdate is a GORM hook impl
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

func (User) TableName() string {
	return "users"
}

// AssertAuth asserts a user can pass authentification.
func (u *User) AssertAuth(db *gorm.DB, userParams *models.UsersParams) error {
	if db == nil || userParams == nil {
		return errors.New("nil *gorm.DB or *models.UsersParams")
	}
	passwd := ""
	if u.Password != "" {
		passwd = crypt.HashPassword(
			u.Password,
			userParams.GetArgon2Params(),
			userParams.GetPasswordSalt(),
		)
	}
	return db.First(u, "login = ? AND password = ?", u.Login, passwd).Error
}

func (u User) IntoClaims() crypt.JWTDefaultClaims {
	return crypt.JWTDefaultClaims{
		UID: u.ID,
	}
}

func (u User) IsRevoked(timeRef time.Time) bool {
	return u.RevokedAt != nil && u.RevokedAt.Before(timeRef)
}

func NewEmptyUser() *User {
	return &User{}
}

func NewUser(login, password string) *User {
	return &User{
		Login:    login,
		Password: password,
	}
}
