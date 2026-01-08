package services

import (
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/entities/constraints"
	"github.com/monkeydioude/goauth/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Test_I_an_Request_Password_Reset(t *testing.T) {
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

	realmIdMock := "22f989a1-852f-4b04-9875-fb7b5e5b141a"
	// Expect a select on a user with a revoked_at date that we reached.
	// Removing 10s to the time reference
	mock.ExpectQuery(`SELECT \* FROM "realms" WHERE name = \$1 AND "realms"."deleted_at" IS NULL ORDER BY "realms"."id" LIMIT \$2`).
		WithArgs("realm_1", 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow(realmIdMock, "realm_1"),
		)
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE login = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`).
		WithArgs("test_login_1", 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(1),
		)
	mock.ExpectQuery(`SELECT \* FROM "user_actions" WHERE user_id = \$1 AND realm_id = \$2 AND action = \$3 AND validated_at IS NULL ORDER BY "user_actions"."id" LIMIT \$4`).
		WithArgs(1, realmIdMock, `reset-password`, 1)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "user_actions".*RETURNING "id"`).
		WithArgs(1, realmIdMock, entities.UserActionTypePassword, `test`, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()
	out, err := UserActionCreate(gormDB, UserActionCreateIn{
		Login:  "test_login_1",
		Realm:  "realm_1",
		Action: "reset-password",
	},
		func() string { return "test" },
	)
	assert.NoError(t, err)
	assert.Equal(t, out.Data, "test")
	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_I_Can_Validate_Password_Reset_Request(t *testing.T) {
	// Set up sqlmock and a mocked *gorm.DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open sqlmock database: %s", err)
	}
	defer db.Close()
	up := &models.UsersParams{
		PasswdSalt:          []byte(os.Getenv(consts.PASSWD_SALT)),
		Argon2params:        consts.Argon2,
		LoginConstraints:    []constraints.LoginConstraint{},
		PasswordConstraints: []constraints.PasswordConstraint{},
	}

	// Use the sqlmock DB connection in Gorm
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize Gorm with sqlmock: %s", err)
	}
	realmIdMock := "22f989a1-852f-4b04-9875-fb7b5e5b141a"
	mock.ExpectQuery(`SELECT \* FROM "realms" WHERE name = \$1 AND "realms"."deleted_at" IS NULL ORDER BY "realms"."id" LIMIT \$2`).
		WithArgs("realm_1", 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name"}).
				AddRow(realmIdMock, "realm_1"),
		)
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE login = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`).
		WithArgs("test_login_1", 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(1),
		)
	mock.ExpectQuery(`SELECT \* FROM "user_actions" WHERE user_id = \$1 AND realm_id = \$2 AND data = \$3 AND validated_at IS NULL ORDER BY "user_actions"."id" LIMIT \$4`).
		WithArgs(1, realmIdMock, `test`, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "action", "user_id"}).
				AddRow(1, entities.UserActionTypePassword, 1),
		)
	mock.ExpectBegin()
	hashedPassword := "WuMC+ABRzZ6vUwR4IutpHJ0yOUTSvX/m0yD1rnc9mdE="
	mock.ExpectExec(`UPDATE "users" SET "password"=\$1,"updated_at"=\$2 WHERE "users"."deleted_at" IS NULL AND "id" = \$3`).
		WithArgs(hashedPassword, sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "user_actions" SET "validated_at"=\$1,"updated_at"=\$2 WHERE "id" = \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	err = UserActionValidate(gormDB, up, UserActionValidateIn{
		Login:   "test_login_1",
		Realm:   "realm_1",
		Data:    "test",
		Against: "test-password",
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// func Test_I_Can_Check_User_Action_Status(t *testing.T) {
// 	// Set up sqlmock and a mocked *gorm.DB
// 	db, mock, err := sqlmock.New()
// 	if err != nil {
// 		t.Fatalf("Failed to open sqlmock database: %s", err)
// 	}
// 	defer db.Close()
// 	// Use the sqlmock DB connection in Gorm
// 	gormDB, err := gorm.Open(postgres.New(postgres.Config{
// 		Conn: db,
// 	}), &gorm.Config{})
// 	if err != nil {
// 		t.Fatalf("Failed to initialize Gorm with sqlmock: %s", err)
// 	}
// 	realmIdMock := "22f989a1-852f-4b04-9875-fb7b5e5b141a"
// 	mock.ExpectQuery(`SELECT \* FROM "realms" WHERE name = \$1 AND "realms"."deleted_at" IS NULL ORDER BY "realms"."id" LIMIT \$2`).
// 		WithArgs("realm_1", 1).
// 		WillReturnRows(
// 			sqlmock.NewRows([]string{"id", "name"}).
// 				AddRow(realmIdMock, "realm_1"),
// 		)
// 	mock.ExpectQuery(`SELECT \* FROM "users" WHERE login = \$1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$2`).
// 		WithArgs("test_login_1", 1).
// 		WillReturnRows(
// 			sqlmock.NewRows([]string{"id"}).AddRow(1),
// 		)
// 	mockDate := time.Date(2026, 1, 8, 11, 32, 30, 0, time.UTC)
// 	mock.ExpectQuery(`SELECT \* FROM "user_actions" WHERE user_id = \$1 AND realm_id = \$2 AND action = \$3 ORDER BY "user_actions"."id" DESC LIMIT \$4`).
// 		WithArgs(1, realmIdMock, entities.UserActionTypeActivate, 1).
// 		WillReturnRows(
// 			sqlmock.NewRows([]string{"action", "validated_at"}).
// 				AddRow(entities.UserActionTypeActivate.String(), mockDate),
// 		)
// 	res, err := UserActionStatuses(gormDB, UserActionStatusIn{
// 		Login:  "test_login_1",
// 		Realm:  "realm_1",
// 		Action: entities.UserActionTypeActivate.String(),
// 	})
// 	assert.NoError(t, err)
// 	assert.Equal(t, entities.UserActionTypeActivate.String(), res.Action)
// 	assert.NotNil(t, res.ValidatedAt)
// 	assert.Equal(t, mockDate.Unix(), res.ValidatedAt.Unix())
// }
