package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"

	"github.com/monkeydioude/goauth/v2/internal/config/boot"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/slice"
)

// realmKind is the kind of the realm to create. It cannot change afterwards.
var realmKind = flag.String("kind", entities.RealmKindHuman, "kind of the realm to create: human or service")

func realmCreate() error {
	args := flag.Args()
	if len(args) < 4 {
		return errors.New("missing realm args (realm create <Allow New User=0|1> <Name of the realm> <Description (optional)>, with -kind=human|service before them)")
	}
	if *realmKind != entities.RealmKindHuman && *realmKind != entities.RealmKindService {
		return fmt.Errorf("unknown realm kind %q: human or service", *realmKind)
	}
	res := boot.PostgreSQLBoot(entities.Realm{})
	if res.IsErr() {
		return res.Error
	}
	db := res.Result()
	realm := entities.Realm{Kind: *realmKind}
	realm.AllowNewUser = args[2] == "1"
	slice.MapVars(args[3:], &realm.Name, &realm.Description)
	return db.Create(&realm).Error
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
		slog.Info(fmt.Sprintf("%d - %s", it+1, realm.Name), "ID", realm.ID, "kind", realm.Kind, "desc", realm.Description)
	}
	return err
}
