package entities

import (
	"GOAuTh/internal/domain/models"
	"GOAuTh/pkg/crypt"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Login        string         `gorm:"unique;not null" json:"login" validator:""`
	Password     string         `gorm:"not null" json:"password,omitempty" validator:""`
	RealmID      uuid.UUID      `gorm:"index"`
	Realm        *Realm         `gorm:"foreignKey:RealmID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	LastLoggedAt *time.Time     `json:"last_logged_at"`
	RevokedAt    *time.Time     `json:"revoked_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// payload only
	RealmName string `gorm:"-" json:"realm_name"`
}

// BeforeCreate is a GORM hook impl
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if tx == nil {
		slog.Error("nil *gorm.DB")
		return errors.New("nil *gorm.DB")
	}
	if u.Login == "" || u.Password == "" || u.RealmName == "" {
		slog.Error("login, password or realm_name cannot be empty")
		return errors.New("login, password or realm_name cannot be empty")
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
	return db.
		Joins("JOIN realms ON realms.id = users.realm_id").
		Preload("Realm").
		First(u, "login = ? AND password = ? AND realms.name = ?", u.Login, passwd, u.RealmName).Error
}

func (u User) IntoClaims() crypt.JWTDefaultClaims {
	return crypt.JWTDefaultClaims{
		UID:   u.ID,
		Realm: u.RealmName,
	}
}

func (u User) IsRevoked(timeRef time.Time) bool {
	return u.RevokedAt != nil && u.RevokedAt.Before(timeRef)
}

func NewEmptyUser() *User {
	return &User{}
}

func NewUser(login, password, realm string) *User {
	return &User{
		Login:     login,
		Password:  password,
		RealmName: realm,
	}
}
