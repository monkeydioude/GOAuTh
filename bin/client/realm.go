package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/monkeydioude/goauth/v2/internal/config/boot"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/slice"
)

// realmKind is the kind of the realm to create. It cannot change afterwards.
var realmKind = flag.String("kind", entities.RealmKindHuman, "kind of the realm to create: human or service")

// realmMaxKeys caps the live access keys an account of the realm to create may hold.
var realmMaxKeys = flag.Int("max-keys", 0, "live access keys an account of the realm may hold; 0 inherits ACCESS_KEY_MAX_ACTIVE")

// accessKeyCapOf turns a cap argument into the realm's value: 0 is nil, which
// inherits the server's. A value above ACCESS_KEY_MAX_ACTIVE is kept but warned
// about: the server's wins until it is raised.
func accessKeyCapOf(value int) (*int, error) {
	if value == 0 {
		return nil, nil
	}
	if value < 0 {
		return nil, fmt.Errorf("access key cap must be at least 1, or 0 to inherit: %d", value)
	}
	if env, err := boot.AccessKeyBoot(); err == nil && value > env.MaxActive {
		slog.Warn("cap above ACCESS_KEY_MAX_ACTIVE, which wins until raised", "cap", value, "ACCESS_KEY_MAX_ACTIVE", env.MaxActive)
	}
	return &value, nil
}

func realmCreate() error {
	args := flag.Args()
	if len(args) < 4 {
		return errors.New("missing realm args (realm create <Allow New User=0|1> <Name of the realm> <Description (optional)>, with -kind=human|service and -max-keys=N before them)")
	}
	if *realmKind != entities.RealmKindHuman && *realmKind != entities.RealmKindService {
		return fmt.Errorf("unknown realm kind %q: human or service", *realmKind)
	}
	maxKeys, err := accessKeyCapOf(*realmMaxKeys)
	if err != nil {
		return err
	}
	res := boot.PostgreSQLBoot(entities.Realm{})
	if res.IsErr() {
		return res.Error
	}
	db := res.Result()
	realm := entities.Realm{Kind: *realmKind, AccessKeyMaxActive: maxKeys}
	realm.AllowNewUser = args[2] == "1"
	slice.MapVars(args[3:], &realm.Name, &realm.Description)
	return db.Create(&realm).Error
}

// realmSetMaxKeys sets how many live access keys an account of the realm may
// hold: a number, or "default" to inherit ACCESS_KEY_MAX_ACTIVE. Lowering it
// revokes nothing; it only stops new keys until the account is under it.
func realmSetMaxKeys() error {
	args := flag.Args()
	if len(args) < 4 {
		return errors.New("missing realm args (realm set-max-keys <Name of the realm> <n|default>)")
	}
	value := 0
	if args[3] != "default" {
		n, err := strconv.Atoi(args[3])
		if err != nil || n < 1 {
			return fmt.Errorf("access key cap must be a number of at least 1, or default: %q", args[3])
		}
		value = n
	}
	maxKeys, err := accessKeyCapOf(value)
	if err != nil {
		return err
	}
	res := boot.PostgreSQLBoot(entities.Realm{})
	if res.IsErr() {
		return res.Error
	}
	tx := res.Result().Model(&entities.Realm{}).Where("name = ?", args[2]).Update("access_key_max_active", maxKeys)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("no realm named %q", args[2])
	}
	return nil
}

func realmsShow() error {
	res := boot.PostgreSQLBoot(entities.Realm{})
	if res.IsErr() {
		return res.Error
	}
	db := res.Result()
	realms := []entities.Realm{}

	err := db.Select("*").Find(&realms).Error
	for it, realm := range realms {
		maxKeys := "default"
		if realm.AccessKeyMaxActive != nil {
			maxKeys = strconv.Itoa(*realm.AccessKeyMaxActive)
		}
		slog.Info(fmt.Sprintf("%d - %s", it+1, realm.Name), "ID", realm.ID, "kind", realm.Kind, "max_keys", maxKeys, "desc", realm.Description)
	}
	return err
}
