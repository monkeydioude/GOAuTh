package services

import (
	"GOAuTh/pkg/crypt"
	entity "GOAuTh/pkg/domain/entities"
	"log"
	"time"

	"gorm.io/gorm"
)

func IsLoginRevoked[C crypt.JWTClaims, U entity.User[C]](
	login string,
	db *gorm.DB,
	timeRef time.Time,
) (bool, error) {
	var user U
	res := db.Where("login = ?", login).First(&user)
	if res.Error != nil {
		log.Printf("[WARN] in isLoginRevoked: %s\n", res.Error.Error())
		return false, res.Error
	}
	return user.IsRevoked(timeRef), nil
}
