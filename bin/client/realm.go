package main

import (
	"GOAuTh/internal/config/boot"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/data_types/slice"
	"errors"
	"flag"
	"fmt"
	"log/slog"
)

func realmCreate() error {
	args := flag.Args()
	if len(args) < 4 {
		return errors.New("missing realm args (realm create <Allow New User=0|1> <Name of the realm> <Description (optional)>)")
	}
	res := boot.PostgreSQLBoot(entities.Realm{})
	if res.IsErr() {
		return res.Error
	}
	db := res.Result()
	realm := entities.Realm{}
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
		slog.Info(fmt.Sprintf(`%d - ID="%s" Name="%s" Desc="%s"`+"\n", it+1, realm.ID, realm.Name, realm.Description))
	}
	return err
}
