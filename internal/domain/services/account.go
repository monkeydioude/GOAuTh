package services

import (
	stdErr "errors"
	"github.com/monkeydioude/goauth/v2/internal/config/logs"
	"log/slog"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AccountCreateIn struct {
	Realm string
	Login string
	// who asked for it, as the consumer names them
	Actor string
}

// liveLoginConflict is a clash with the login of one of the realm's live accounts
// (idx_realm_login_active): a deleted account's login is free again.
var liveLoginConflict = clause.OnConflict{
	Columns:     []clause.Column{{Name: "realm_id"}, {Name: "login"}},
	TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
	DoNothing:   true,
}

// AccountCreate makes an account in a realm for a trusted backend, the way the
// realm's kind says. It is idempotent on (realm, login): when the realm already
// has a live account of this login, that account comes back as it is and created
// is false, so a consumer that lost the id of the account it made can ask again.
// The account comes back with its realm loaded.
func AccountCreate(db *gorm.DB, in AccountCreateIn) (account *entities.User, created bool, err error) {
	if db == nil {
		return nil, false, errors.InternalServerError(stdErr.New("nil db object"))
	}
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		if stdErr.Is(err, gorm.ErrRecordNotFound) {
			slog.ErrorContext(logs.DBContext(db), consts.ERR_REALM_NOT_FOUND, "realm_name", in.Realm)
			return nil, false, errors.NotFound(stdErr.New(consts.ERR_REALM_NOT_FOUND))
		}
		return nil, false, errors.DBError(err)
	}
	account, err = KindOf(realm).NewAccount(realm, in.Login, in.Actor)
	if err != nil {
		return nil, false, err
	}
	// in one statement, so two calls at once still make a single account
	inserted := db.Clauses(liveLoginConflict).Create(account)
	if inserted.Error != nil {
		return nil, false, errors.DBError(inserted.Error)
	}
	if inserted.RowsAffected == 0 {
		return liveAccount(db, realm, in.Login)
	}
	account.Realm = &realm
	return account, true, nil
}

// liveAccount is the realm's live account of this login, which an insert just clashed with.
func liveAccount(db *gorm.DB, realm entities.Realm, login string) (*entities.User, bool, error) {
	var account entities.User
	if err := db.Where("login = ? AND realm_id = ?", login, realm.ID).First(&account).Error; err != nil {
		return nil, false, errors.DBError(err)
	}
	slog.InfoContext(logs.DBContext(db), "account already exists, returned as is", "account_id", account.ID, "realm_name", realm.Name)
	account.Realm = &realm
	return &account, false, nil
}
