package services

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/pkg/crypt"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestJWTRefreshLogsNoTokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open sqlmock database: %s", err)
	}
	defer db.Close()
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize Gorm with sqlmock: %s", err)
	}

	var logs bytes.Buffer
	defaultOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(defaultOutput) })

	timeRefFn := func() time.Time {
		return time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	}
	revocCheckerFn := func(uint, func() time.Time) (bool, error) {
		return false, nil
	}
	atf := NewJWTFactory(crypt.HS256("test"), time.Hour, timeRefFn, revocCheckerFn, "Authorization")
	rtf := NewJWTFactory(crypt.HS256("test"), 24*time.Hour, timeRefFn, revocCheckerFn, "Refresh")
	refreshToken, err := rtf.GenerateToken(crypt.JWTDefaultClaims{UID: 1, Realm: "test"})
	assert.NoError(t, err)

	// the stored token differs: the mismatch is logged without either token
	mock.ExpectQuery(`SELECT .* FROM "users" WHERE id = \$1`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"refresh_token"}).AddRow("stored-refresh-token"))
	_, _, err = JWTRefresh(refreshToken.Token, *atf, *rtf, gormDB)
	assert.Error(t, err)

	// the stored token matches: the refresh succeeds without logging any token
	mock.ExpectQuery(`SELECT .* FROM "users" WHERE id = \$1`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"refresh_token"}).AddRow(refreshToken.Token))
	accessCookie, refreshCookie, err := JWTRefresh(refreshToken.Token, *atf, *rtf, gormDB)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
	for _, secret := range []string{refreshToken.Token, "stored-refresh-token", accessCookie.Value, refreshCookie.Value} {
		assert.NotContains(t, logs.String(), secret)
	}
}
