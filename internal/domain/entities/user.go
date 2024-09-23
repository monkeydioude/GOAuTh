package entities

import (
	"GOAuTh/internal/domain/models"
	"GOAuTh/pkg/crypt"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"-"`
	Login        string         `gorm:"unique;not null" json:"login"`
	Password     string         `gorm:"not null" json:"password,omitempty"`
	CreatedAt    time.Time      `json:"-"`
	UpdatedAt    time.Time      `json:"-"`
	LastLoggedAt *time.Time     `json:"-"`
	RevokedAt    *time.Time     `json:"-"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate is a GORM hook impl
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Login == "" || u.Password == "" {
		return errors.New("login or password cannot be empty")
	}
	fmt.Println(u.Password)
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
		Name: u.Login,
	}
}

func (u User) IsRevoked(timeRef time.Time) bool {
	return u.RevokedAt != nil && u.RevokedAt.Before(timeRef)
}

func NewUser() *User {
	return &User{}
}
