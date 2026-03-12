package services

import (
	"errors"
	"log"
	"time"

	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/domain/entities"

	"gorm.io/gorm"
)

func IsLoginRevoked[C crypt.JWTClaims, U entities.User[C]](
	uid uint,
	db *gorm.DB,
	timeRef time.Time,
) (bool, error) {
	if db == nil {
		return false, errors.New("nil *gorm.DB")
	}
	var user U
	res := db.Where("id = ?", uid).First(&user)
	if res.Error != nil {
		log.Printf("[WARN] in isLoginRevoked: %s\n", res.Error.Error())
		return false, res.Error
	}
	return user.IsRevoked(timeRef), nil
}
