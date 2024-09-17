package functional_tests

import (
	"GOAuTh/internal/boot"
	"GOAuTh/internal/consts"
	"GOAuTh/internal/entities"
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/constraints"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _gorm *gorm.DB = nil

func setup(t *testing.T) (*handlers.Layout, *gorm.DB, time.Time) {
	if os.Getenv("DB_PATH") == "" {
		fmt.Println("[INFO] no DB_PATH env found. Fallback on postgres://test:test@0.0.0.0:5445/test_db (make run-test-db)")
		os.Setenv("DB_PATH", "postgres://test:test@0.0.0.0:5445/test_db")
	}
	os.Setenv("JWT_SECRET", "test")
	var err error
	var layout *handlers.Layout

	// init layout
	if res := boot.LayoutBoot([]any{entities.NewDefaultUser()}, constraints.EmailConstraint); res.IsErr() {
		panic("Could not boot layout")
	} else {
		layout = res.Result()
	}

	var gormDB *gorm.DB = nil
	if _gorm == nil {
		// Use the sqlmock DB connection in Gorm
		_gorm, err = gorm.Open(postgres.Open(fmt.Sprintf("%s?search_path=%s", os.Getenv(consts.DB_PATH), "public")), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to initialize Gorm: %s", err)
		}
	}
	gormDB = _gorm

	timeRef := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	// change to the JWTFactory, so we can manipulate
	// its time reference logic
	layout.JWTFactory.TimeFn = func() time.Time {
		return timeRef
	}
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	if t != nil && _gorm != nil {
		t.Cleanup(func() {
			sql, _ := _gorm.DB()
			sql.Close()
			_gorm = nil
		})
	}

	return layout, gormDB, timeRef
}
