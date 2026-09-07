package services

import (
	"fmt"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"gorm.io/gorm"
)

type GetsUsersInfoIn struct {
	Login *string
	Realm *string
}

func GetUsersInfo(in *GetsUsersInfoIn, db *gorm.DB) ([]entities.User, error) {
	var users []entities.User
	tx := db
	if in.Login != nil && *in.Login != "" {
		tx = tx.Where("login = ?", *in.Login)
	}
	if in.Realm != nil && *in.Realm != "" {
		tx = tx.Where("realm_id IN (SELECT id FROM realms WHERE name = ?)", *in.Realm)
	}
	err := tx.Preload("Realm").Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %v", err)
	}
	return users, nil
}
