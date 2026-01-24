package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/crypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func dbBoot() *gorm.DB {
	godotenv.Load()
	schema := os.Getenv(consts.DB_SCHEMA)
	if schema == "" {
		schema = "public"
	}
	db, err := gorm.Open(postgres.Open(fmt.Sprintf("%s?search_path=%s", os.Getenv(consts.DB_PATH), schema)))
	if err != nil {
		panic(err)
	}
	return db
}

func main() {
	db := dbBoot()
	flag.Parse()
	args := flag.Args()
	factory := &services.JWTFactory{
		SigningMethod:       crypt.HS256(os.Getenv(consts.JWT_SECRET)),
		ExpiresIn:           consts.JWTExpiresIn,
		RefreshesIn:         consts.JWTRefreshesIn,
		TimeFn:              func() time.Time { return time.Now() },
		RevocationCheckerFn: func(uid uint, timeFn func() time.Time) (bool, error) { return false, nil },
	}
	if len(args) != 3 {
		slog.Error("invalid args. Must be 3 args: <expire date YYYY-MM-DD> <email> <realm>", "length", len(args))
		return
	}
	expRef, err := time.Parse(time.DateOnly, args[0])
	if err != nil {
		slog.Error("time.Parse error", "err", err.Error())
		return
	}
	user := entities.User{}
	err = db.Joins("JOIN realms ON realms.id = users.realm_id").Where("users.login = ? AND realms.name = ?", args[1], args[2]).Find(&user).Error
	if err != nil {
		slog.Error("db.Where error", "err", err.Error())
		return
	}
	token, err := factory.GenerateToken(crypt.JWTDefaultClaims{
		Expire:  expRef.Unix(),
		Refresh: expRef.Unix(),
		UID:     user.ID,
		Realm:   args[2],
	})
	if err != nil {
		slog.Error("factory.GenerateToken error", "err", err.Error())
		return
	}
	fmt.Fprintln(os.Stdout, token)
}
