package services

import (
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/crypt"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
)

func TestFindUserByLogin(t *testing.T) {
	// Set up sqlmock and a mocked *gorm.DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open sqlmock database: %s", err)
	}
	defer db.Close()

	// Use the sqlmock DB connection in Gorm
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize Gorm with sqlmock: %s", err)
	}
	timeRef := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)

	// Expect a select on a user with no revoked_at date
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE login = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`).
		WithArgs("user_1_false", 1). // Expecting both "login" and the "LIMIT" value
		WillReturnRows(sqlmock.NewRows([]string{"id", "login", "revoked_at"}).
			AddRow(1, "user_1_false", nil))

	isRevoked, err := IsLoginRevoked[crypt.JWTDefaultClaims, entities.User]("user_1_false", gormDB, timeRef)
	if err != nil {
		t.Error(err.Error())
	}
	if isRevoked == true {
		t.Error("user_1_false: isRevoked should be false")
	}

	// Expect a select on a user with a revoked_at date that we still haven't reached
	// Adding 10s to the time reference
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE login = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`).
		WithArgs("user_2_false", 1). // Expecting both "login" and the "LIMIT" value
		WillReturnRows(sqlmock.NewRows([]string{"id", "login", "revoked_at"}).
			AddRow(1, "user_2_false", timeRef.Add(10*time.Second)))
	isRevoked, err = IsLoginRevoked[crypt.JWTDefaultClaims, entities.User]("user_2_false", gormDB, timeRef)
	if err != nil {
		t.Error(err.Error())
	}
	if isRevoked == true {
		t.Error("user_2_false: isRevoked should be false")
	}

	// Expect a select on a user with a revoked_at date that we reached.
	// Removing 10s to the time reference
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE login = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`).
		WithArgs("user_3_true", 1). // Expecting both "login" and the "LIMIT" value
		WillReturnRows(sqlmock.NewRows([]string{"id", "login", "revoked_at"}).
			AddRow(1, "user_3_true", timeRef.Add(-10*time.Second)))
	isRevoked, err = IsLoginRevoked[crypt.JWTDefaultClaims, entities.User]("user_3_true", gormDB, timeRef)
	if err != nil {
		t.Error(err.Error())
	}
	if isRevoked == false {
		t.Error("user_3_true: isRevoked should be true")
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}