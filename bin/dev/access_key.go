package main

import (
	"fmt"
	"log"
	"os/user"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/boot"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"

	"gorm.io/gorm"
)

// accessKeyCreate mints a key that never expires for the account login of the
// realm, making the account first when the realm has none by that login. It
// prints the key, this once, and the ids to find or revoke it by.
func accessKeyCreate(realm, login string) {
	keyEnv, err := boot.AccessKeyBoot()
	if err != nil {
		log.Fatal(err)
	}
	// no entities: the API binary owns the migrations
	dbRes := boot.PostgreSQLBoot()
	if dbRes.IsErr() {
		log.Fatal(dbRes.Error)
	}
	db := dbRes.Result()
	actor := operator()

	account, err := accountOf(db, realm, login, actor)
	if err != nil {
		log.Fatalf("could not get the account %s of realm %s: %s", login, realm, err)
	}
	key, row, err := services.AccessKeyCreate(db, services.AccessKeyCreateIn{
		AccountID: account.ID,
		Realm:     realm,
		Name:      login,
		Actor:     actor,
	}, keyEnv.MaxActive, time.Now())
	if err != nil {
		log.Fatalf("could not create a key for the account %s (%d): %s", login, account.ID, err)
	}
	fmt.Printf("key, shown this once: %s\n", key)
	fmt.Printf("key_id:               %s\n", row.ID)
	fmt.Printf("account_id:           %d\n", account.ID)
}

// accountOf is the account login of the realm, made on behalf of actor when
// there is none yet.
func accountOf(db *gorm.DB, realm, login, actor string) (*entities.User, error) {
	var account entities.User
	// Find, not First: a miss is expected here, not an error to log
	res := db.Joins("JOIN realms ON realms.id = users.realm_id AND realms.deleted_at IS NULL").
		Where("users.login = ? AND realms.name = ?", login, realm).
		Limit(1).
		Find(&account)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected > 0 {
		return &account, nil
	}
	created, _, err := services.AccountCreate(db, services.AccountCreateIn{Realm: realm, Login: login, Actor: actor})
	if err != nil {
		return nil, err
	}
	log.Printf("account %s created in realm %s", login, realm)
	return created, nil
}

// operator is who goauth records as the key's creator: whoever runs this.
func operator() string {
	if current, err := user.Current(); err == nil {
		return "cli:" + current.Username
	}
	return "cli"
}
