package services

import (
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities/constraints"
	"github.com/monkeydioude/goauth/v2/internal/domain/models"
	"github.com/monkeydioude/goauth/v2/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const realmIdMock = "22f989a1-852f-4b04-9875-fb7b5e5b141a"

var timeRef = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

// clockAt is a clock stopped at at.
func clockAt(at time.Time) func() time.Time {
	return func() time.Time { return at }
}

// newMockDB is a *gorm.DB on sqlmock.
func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open sqlmock database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize Gorm with sqlmock: %s", err)
	}
	return gormDB, mock
}

func testUsersParams() *models.UsersParams {
	return &models.UsersParams{
		PasswdSalt:          []byte(os.Getenv(consts.PASSWD_SALT)),
		Argon2params:        consts.Argon2,
		LoginConstraints:    []constraints.LoginConstraint{},
		PasswordConstraints: []constraints.PasswordConstraint{},
	}
}

// expectRealmAndUser: realm_1 and its user test_login_1, of id 1.
func expectRealmAndUser(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`SELECT \* FROM "realms" WHERE name = \$1 AND "realms"."deleted_at" IS NULL ORDER BY "realms"."id" LIMIT \$2`).
		WithArgs("realm_1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(realmIdMock, "realm_1"))
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE \(login = \$1 AND realm_id = \$2\) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT \$3`).
		WithArgs("test_login_1", realmIdMock, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "login"}).AddRow(1, "test_login_1"))
}

// expectPendingByAction: the user's pending verify-email action, found by its type.
func expectPendingByAction(mock sqlmock.Sqlmock, rows *sqlmock.Rows) {
	mock.ExpectQuery(`SELECT \* FROM "user_actions" WHERE user_id = \$1 AND realm_id = \$2 AND action = \$3 AND validated_at IS NULL ORDER BY "user_actions"."id" DESC LIMIT \$4`).
		WithArgs(1, realmIdMock, "verify-email", 1).
		WillReturnRows(rows)
}

// pendingCode is a verify-email action of code 123456, with these tries and expiry.
func pendingCode(attempts, maxAttempts int, expiresAt *time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "user_id", "action", "data", "attempts", "max_attempts", "expires_at"}).
		AddRow(7, 1, "verify-email", "123456", attempts, maxAttempts, expiresAt)
}

const useATrySQL = `UPDATE "user_actions" SET "attempts"=attempts \+ 1 WHERE id = \$1 AND \(max_attempts = 0 OR attempts < max_attempts\)`

func validateCode(gormDB *gorm.DB, code string, now time.Time) (string, error) {
	return UserActionValidate(gormDB, testUsersParams(), UserActionValidateIn{
		Realm:  "realm_1",
		Login:  "test_login_1",
		Action: "verify-email",
		Data:   code,
	}, clockAt(now))
}

func Test_I_an_Request_Password_Reset(t *testing.T) {
	gormDB, mock := newMockDB(t)
	expectRealmAndUser(mock)
	mock.ExpectQuery(`SELECT \* FROM "user_actions" WHERE user_id = \$1 AND realm_id = \$2 AND action = \$3 AND validated_at IS NULL ORDER BY "user_actions"."id" LIMIT \$4`).
		WithArgs(1, realmIdMock, `reset-password`, 1)
	mock.ExpectBegin()
	// no expiry, no try limit: as before #155
	mock.ExpectQuery(`INSERT INTO "user_actions".*RETURNING "id"`).
		WithArgs(1, realmIdMock, entities.UserActionTypePassword, `test`, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil, 0, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()
	out, err := UserActionCreate(gormDB, UserActionCreateIn{
		Login:  "test_login_1",
		Realm:  "realm_1",
		Action: "reset-password",
	},
		func() string { return "test" },
		clockAt(timeRef),
	)
	assert.NoError(t, err)
	assert.Equal(t, out.Data, "test")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_I_Can_Create_An_Action_That_Expires_With_A_Try_Limit(t *testing.T) {
	gormDB, mock := newMockDB(t)
	expectRealmAndUser(mock)
	mock.ExpectQuery(`SELECT \* FROM "user_actions" WHERE user_id = \$1 AND realm_id = \$2 AND action = \$3 AND validated_at IS NULL ORDER BY "user_actions"."id" LIMIT \$4`).
		WithArgs(1, realmIdMock, "verify-email", 1)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "user_actions".*RETURNING "id"`).
		WithArgs(1, realmIdMock, "verify-email", "123456", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), timeRef.Add(30*time.Minute), 0, 5).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()
	code := "123456"

	_, err := UserActionCreate(gormDB, UserActionCreateIn{
		Login:       "test_login_1",
		Realm:       "realm_1",
		Action:      "verify-email",
		Data:        &code,
		TTL:         30 * time.Minute,
		MaxAttempts: 5,
	}, func() string { return "unused" }, clockAt(timeRef))

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_I_Cannot_Create_An_Action_With_Negative_Limits(t *testing.T) {
	for name, in := range map[string]UserActionCreateIn{
		"ttl":          {TTL: -time.Second},
		"max attempts": {MaxAttempts: -1},
	} {
		t.Run(name, func(t *testing.T) {
			gormDB, mock := newMockDB(t)

			_, err := UserActionCreate(gormDB, in, func() string { return "test" }, clockAt(timeRef))

			assert.Equal(t, 422, err.(errors.Err).Code())
			assert.NoError(t, mock.ExpectationsWereMet(), "refused before any query")
		})
	}
}

func Test_I_Can_Validate_Password_Reset_Request(t *testing.T) {
	gormDB, mock := newMockDB(t)
	expectRealmAndUser(mock)
	mock.ExpectQuery(`SELECT \* FROM "user_actions" WHERE user_id = \$1 AND realm_id = \$2 AND data = \$3 AND validated_at IS NULL ORDER BY "user_actions"."id" LIMIT \$4`).
		WithArgs(1, realmIdMock, `test`, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "action", "user_id"}).
				AddRow(1, entities.UserActionTypePassword, 1),
		)
	// the action is claimed, then the password reset, in one transaction
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "user_actions" SET "updated_at"=\$1,"validated_at"=\$2 WHERE \(id = \$3 AND validated_at IS NULL AND \(expires_at IS NULL OR expires_at > \$4\)\) AND \(\(max_attempts = 0 OR attempts < max_attempts\)\)`).
		WithArgs(timeRef, timeRef, 1, timeRef).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SAVEPOINT sp\d+`).WillReturnResult(sqlmock.NewResult(0, 0))
	hashedPassword := "WuMC+ABRzZ6vUwR4IutpHJ0yOUTSvX/m0yD1rnc9mdE="
	mock.ExpectExec(`UPDATE "users" SET "password"=\$1,"updated_at"=\$2 WHERE "users"."deleted_at" IS NULL AND "id" = \$3`).
		WithArgs(hashedPassword, sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// in the same transaction, every session of the user is revoked
	mock.ExpectExec(`UPDATE "sessions" SET "deleted_at"=\$1,"revoked_reason"=\$2 WHERE user_id = \$3 AND "sessions"."deleted_at" IS NULL`).
		WithArgs(sqlmock.AnyArg(), entities.SessionRevokedPasswordReset, 1).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	resEmail, err := UserActionValidate(gormDB, testUsersParams(), UserActionValidateIn{
		Realm:   "realm_1",
		Data:    "test",
		Against: "test-password",
		Login:   "test_login_1",
	}, clockAt(timeRef))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Equal(t, "test_login_1", resEmail)
}

func Test_A_Wrong_Code_Uses_A_Try(t *testing.T) {
	gormDB, mock := newMockDB(t)
	expectRealmAndUser(mock)
	expectPendingByAction(mock, pendingCode(2, 5, nil))
	mock.ExpectBegin()
	mock.ExpectExec(useATrySQL).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	_, err := validateCode(gormDB, "000000", timeRef)

	assert.Equal(t, 404, err.(errors.Err).Code())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_A_Wrong_Code_Without_Tries_Left_Is_Gone(t *testing.T) {
	gormDB, mock := newMockDB(t)
	expectRealmAndUser(mock)
	// read with a try left, but parallel wrong codes used it meanwhile
	expectPendingByAction(mock, pendingCode(4, 5, nil))
	mock.ExpectBegin()
	mock.ExpectExec(useATrySQL).WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	_, err := validateCode(gormDB, "000000", timeRef)

	assert.Equal(t, 410, err.(errors.Err).Code())
	assert.Equal(t, consts.ERR_USER_ACTION_TOO_MANY_ATTEMPTS, err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_An_Action_Is_Refused_Once_Expired_Or_Its_Tries_Used_Up(t *testing.T) {
	expired := timeRef.Add(-time.Second)
	tests := map[string]struct {
		rows *sqlmock.Rows
		want string
	}{
		"expired":       {rows: pendingCode(0, 5, &expired), want: consts.ERR_USER_ACTION_EXPIRED},
		"tries used up": {rows: pendingCode(5, 5, nil), want: consts.ERR_USER_ACTION_TOO_MANY_ATTEMPTS},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gormDB, mock := newMockDB(t)
			expectRealmAndUser(mock)
			expectPendingByAction(mock, tt.rows)

			// even the right code: nothing is written
			_, err := validateCode(gormDB, "123456", timeRef)

			assert.Equal(t, 410, err.(errors.Err).Code())
			assert.Equal(t, tt.want, err.Error())
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func Test_User_Action_Expiry_And_Tries(t *testing.T) {
	at := timeRef.Add(30 * time.Minute)
	action := entities.UserAction{ExpiresAt: &at}
	assert.False(t, action.Expired(at.Add(-time.Nanosecond)))
	assert.True(t, action.Expired(at), "expired from expires_at on")
	assert.False(t, entities.UserAction{}.Expired(timeRef.AddDate(100, 0, 0)), "no expires_at: never")

	assert.False(t, entities.UserAction{Attempts: 4, MaxAttempts: 5}.TriesUsedUp())
	assert.True(t, entities.UserAction{Attempts: 5, MaxAttempts: 5}.TriesUsedUp())
	assert.False(t, entities.UserAction{Attempts: 1000}.TriesUsedUp(), "max_attempts 0: no limit")

	assert.Nil(t, expiresAt(timeRef, 0))
	assert.Equal(t, at, *expiresAt(timeRef, 30*time.Minute))
}
