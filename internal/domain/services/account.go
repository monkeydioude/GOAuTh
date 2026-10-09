package services

import (
	stdErr "errors"
	"github.com/monkeydioude/goauth/v2/internal/config/logs"
	"log/slog"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"gorm.io/gorm"
)

type AccountCreateIn struct {
	Realm string
	Login string
	// who asked for it, as the consumer names them
	Actor string
}

// AccountCreate makes an account in a realm for a trusted backend, the way the
// realm's kind says. The account comes back with its realm loaded.
func AccountCreate(db *gorm.DB, in AccountCreateIn) (*entities.User, error) {
	if db == nil {
		return nil, errors.InternalServerError(stdErr.New("nil db object"))
	}
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		if stdErr.Is(err, gorm.ErrRecordNotFound) {
			slog.ErrorContext(logs.DBContext(db), consts.ERR_REALM_NOT_FOUND, "realm_name", in.Realm)
			return nil, errors.NotFound(stdErr.New(consts.ERR_REALM_NOT_FOUND))
		}
		return nil, errors.DBError(err)
	}
	account, err := KindOf(realm).NewAccount(realm, in.Login, in.Actor)
	if err != nil {
		return nil, err
	}
	// a login is unique within its realm
	var taken int64
	if err := db.Model(&entities.User{}).Where("login = ? AND realm_id = ?", in.Login, realm.ID).Count(&taken).Error; err != nil {
		return nil, errors.DBError(err)
	}
	if taken > 0 {
		slog.ErrorContext(logs.DBContext(db), consts.ERR_USER_ALREADY_EXIST, "login", in.Login, "realm_name", in.Realm)
		return nil, errors.Conflict(stdErr.New(consts.ERR_USER_ALREADY_EXIST))
	}
	if err := db.Create(account).Error; err != nil {
		return nil, errors.DBError(err)
	}
	account.Realm = &realm
	return account, nil
}
