package functional

import (
	"context"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func verifyKey(t *testing.T, conn *grpc.ClientConn, key string) *v1.VerifyAccessKeyResponse {
	res, err := v1.NewAccessKeyClient(conn).Verify(context.Background(), &v1.VerifyAccessKeyRequest{Key: key})
	assert.NoError(t, err)
	return res
}

// assertInvalidKey checks key is refused with the one answer every dead key gets.
func assertInvalidKey(t *testing.T, conn *grpc.ClientConn, key string, why string) {
	res := verifyKey(t, conn, key)
	assert.Equal(t, int32(401), res.Code, why)
	assert.Equal(t, consts.ERR_INVALID_KEY, res.Message, why)
	// nothing of the account leaks
	assert.Empty(t, res.KeyId, why)
	assert.Empty(t, res.Login, why)
	assert.Zero(t, res.AccountId, why)
}

func TestAccessKeyVerifyTellsWhoseAKeyIs(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccessKeyVerifyTellsWhoseAKeyIs", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")
	forever := createKey(t, conn, account, realm.Name, "forever", nil)
	expiry := timeRef.Add(time.Hour)
	temporary := createKey(t, conn, account, realm.Name, "temporary", timestamppb.New(expiry))

	res := verifyKey(t, conn, forever.Key)
	assert.Equal(t, int32(200), res.Code)
	assert.Equal(t, forever.Info.KeyId, res.KeyId)
	assert.Equal(t, account, res.AccountId)
	assert.Equal(t, "sb:org:42", res.Login)
	assert.Equal(t, realm.Name, res.Realm)
	assert.Equal(t, entities.RealmKindService, res.RealmKind)
	assert.Nil(t, res.ExpiresAt)

	res = verifyKey(t, conn, temporary.Key)
	assert.Equal(t, int32(200), res.Code)
	assert.Equal(t, temporary.Info.KeyId, res.KeyId)
	assert.Equal(t, expiry.Unix(), res.ExpiresAt.AsTime().Unix())
}

func TestAccessKeyVerifyRefusesEveryDeadKeyTheSameWay(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccessKeyVerifyRefusesEveryDeadKey", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")
	live := createKey(t, conn, account, realm.Name, "live", nil)
	assert.Equal(t, int32(200), verifyKey(t, conn, live.Key).Code)

	// malformed and unknown
	assertInvalidKey(t, conn, "", "empty")
	assertInvalidKey(t, conn, "gak_short", "malformed")
	assertInvalidKey(t, conn, live.Key[:46]+"!", "a key with a wrong character")
	unknown, err := newUnknownKey()
	assert.NoError(t, err)
	assertInvalidKey(t, conn, unknown, "well formed, never created")

	// revoked
	revoked := createKey(t, conn, account, realm.Name, "revoked", nil)
	assert.Equal(t, int32(200), verifyKey(t, conn, revoked.Key).Code)
	assert.Equal(t, int32(200), revokeKey(t, conn, account, realm.Name, revoked.Info.KeyId, "revoked by human").Code)
	assertInvalidKey(t, conn, revoked.Key, "revoked")

	// expired: alive until its expiry, then not
	expiring := createKey(t, conn, account, realm.Name, "expiring", timestamppb.New(timeRef.Add(time.Hour)))
	assert.Equal(t, int32(200), verifyKey(t, conn, expiring.Key).Code)
	layout.AccessTokenFactory.TimeFn = func() time.Time { return timeRef.Add(time.Hour) }
	assertInvalidKey(t, conn, expiring.Key, "expired")
	layout.AccessTokenFactory.TimeFn = func() time.Time { return timeRef }
	assert.Equal(t, int32(200), verifyKey(t, conn, expiring.Key).Code)

	// the owner gone, with the key itself left untouched: the join says no
	assert.NoError(t, gormDB.Delete(&entities.User{}, account).Error)
	assertInvalidKey(t, conn, live.Key, "account deleted")
	assert.False(t, storedKey(t, gormDB, live.Info.KeyId).RevokedAt.Valid)
	assert.NoError(t, gormDB.Unscoped().Model(&entities.User{}).Where("id = ?", account).Update("deleted_at", nil).Error)
	assert.Equal(t, int32(200), verifyKey(t, conn, live.Key).Code)

	// the owner revoked: from a date on, not before
	assert.NoError(t, gormDB.Model(&entities.User{}).Where("id = ?", account).Update("revoked_at", timeRef.Add(time.Minute)).Error)
	assert.Equal(t, int32(200), verifyKey(t, conn, live.Key).Code)
	assert.NoError(t, gormDB.Model(&entities.User{}).Where("id = ?", account).Update("revoked_at", timeRef.Add(-time.Minute)).Error)
	assertInvalidKey(t, conn, live.Key, "account revoked")
	assert.NoError(t, gormDB.Model(&entities.User{}).Where("id = ?", account).Update("revoked_at", nil).Error)
	assert.Equal(t, int32(200), verifyKey(t, conn, live.Key).Code)

	// the realm gone
	assert.NoError(t, gormDB.Delete(&realm).Error)
	assertInvalidKey(t, conn, live.Key, "realm deleted")
}

func TestAccessKeyVerifyRefusesTheKeysOfADeletedAccount(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccessKeyVerifyRefusesTheKeysOfADeleted", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")
	key := createKey(t, conn, account, realm.Name, "one", nil)
	assert.Equal(t, int32(200), verifyKey(t, conn, key.Key).Code)

	actor := "deleted by human"
	res, err := v1.NewAuthClient(conn).Delete(context.Background(), &v1.AuthIdRequest{Uid: account, Actor: &actor})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	assertInvalidKey(t, conn, key.Key, "deleted through Auth.Delete")
}

func TestAccessKeyVerifyNotesTheUseAtMostOncePerMinute(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccessKeyVerifyNotesTheUse", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")
	key := createKey(t, conn, account, realm.Name, "one", nil)
	at := func(moment time.Time) { layout.AccessTokenFactory.TimeFn = func() time.Time { return moment } }
	lastUsed := func() *time.Time { return storedKey(t, gormDB, key.Info.KeyId).LastUsedAt }

	assert.Nil(t, lastUsed())
	// a key that fails verification is not used
	unknown, err := newUnknownKey()
	assert.NoError(t, err)
	assertInvalidKey(t, conn, unknown, "unknown")
	assert.Nil(t, lastUsed())

	// the first use is noted
	assert.Equal(t, int32(200), verifyKey(t, conn, key.Key).Code)
	if assert.NotNil(t, lastUsed()) {
		assert.Equal(t, timeRef.Unix(), lastUsed().Unix())
	}
	// then left alone for a minute
	for _, later := range []time.Duration{time.Second, 30 * time.Second, time.Minute} {
		at(timeRef.Add(later))
		assert.Equal(t, int32(200), verifyKey(t, conn, key.Key).Code)
		assert.Equal(t, timeRef.Unix(), lastUsed().Unix(), "within the window: %s", later)
	}
	// and noted again after it
	at(timeRef.Add(time.Minute + time.Second))
	assert.Equal(t, int32(200), verifyKey(t, conn, key.Key).Code)
	assert.Equal(t, timeRef.Add(time.Minute+time.Second).Unix(), lastUsed().Unix())

	// shown by List
	listed := listKeys(t, conn, account, realm.Name, false).Keys
	if assert.Len(t, listed, 1) && assert.NotNil(t, listed[0].LastUsedAt) {
		assert.Equal(t, timeRef.Add(time.Minute+time.Second).Unix(), listed[0].LastUsedAt.AsTime().Unix())
	}
}
