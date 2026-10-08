package functional

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// newServiceAccount makes an account in a service realm over gRPC.
func newServiceAccount(t *testing.T, conn *grpc.ClientConn, realm string, login string) int32 {
	res, err := v1.NewAccountClient(conn).Create(context.Background(), &v1.CreateAccountRequest{Realm: realm, Login: login, Actor: "human creation"})
	assert.NoError(t, err)
	assert.Equal(t, int32(201), res.Code)
	return res.AccountId
}

func createKey(t *testing.T, conn *grpc.ClientConn, accountID int32, realm string, name string, expiresAt *timestamppb.Timestamp) *v1.CreateAccessKeyResponse {
	res, err := v1.NewAccessKeyClient(conn).Create(context.Background(), &v1.CreateAccessKeyRequest{AccountId: accountID, Realm: realm, Name: name, ExpiresAt: expiresAt, Actor: "human creation"})
	assert.NoError(t, err)
	return res
}

func listKeys(t *testing.T, conn *grpc.ClientConn, accountID int32, realm string, includeRevoked bool) *v1.ListAccessKeysResponse {
	res, err := v1.NewAccessKeyClient(conn).List(context.Background(), &v1.ListAccessKeysRequest{AccountId: accountID, Realm: realm, IncludeRevoked: includeRevoked})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	return res
}

func revokeKey(t *testing.T, conn *grpc.ClientConn, accountID int32, realm string, keyID string, actor string) *v1.Response {
	res, err := v1.NewAccessKeyClient(conn).Revoke(context.Background(), &v1.RevokeAccessKeyRequest{AccountId: accountID, Realm: realm, KeyId: keyID, Actor: actor})
	assert.NoError(t, err)
	return res
}

func storedKey(t *testing.T, gormDB *gorm.DB, keyID string) entities.AccessKey {
	var key entities.AccessKey
	assert.NoError(t, gormDB.Unscoped().First(&key, "id = ?", keyID).Error)
	return key
}

func TestAccessKeysTableIsMigrated(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	m := gormDB.Migrator()
	assert.True(t, m.HasTable(&entities.AccessKey{}))
	for _, field := range []string{"KeyHash", "AccountID", "RevokedAt"} {
		assert.True(t, m.HasIndex(&entities.AccessKey{}, field), "missing index on %s", field)
	}
	assert.True(t, m.HasConstraint(&entities.AccessKey{}, "Account"))
	assert.True(t, m.HasConstraint(&entities.Realm{}, "chk_realms_access_key_max_active"))

	account := newRealmUser(t, gormDB, "TestAccessKeysTableIsMigrated", "TestAccessKeysTableIsMigrated@test.com")
	key := entities.AccessKey{AccountID: account.ID, Name: "one", KeyHash: "hash-1", Prefix: "AAAAAAAA", CreatedAt: timeRef, CreatedBy: "test"}
	assert.NoError(t, gormDB.Create(&key).Error)
	// a hash belongs to one key only
	duplicate := entities.AccessKey{AccountID: account.ID, Name: "two", KeyHash: "hash-1", Prefix: "BBBBBBBB", CreatedAt: timeRef, CreatedBy: "test"}
	assert.Error(t, gormDB.Create(&duplicate).Error)
	// hard-deleting the account deletes its keys
	assert.NoError(t, gormDB.Unscoped().Delete(&account).Error)
	var count int64
	assert.NoError(t, gormDB.Unscoped().Model(&entities.AccessKey{}).Where("account_id = ?", account.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestAccessKeyCreateShowsTheKeyOnceAndKeepsItsHash(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccessKeyCreateShowsTheKeyOnce", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")

	res := createKey(t, conn, account, realm.Name, "Zapier prod", nil)
	assert.Equal(t, int32(201), res.Code)
	assert.Len(t, res.Key, 47)
	assert.True(t, strings.HasPrefix(res.Key, "gak_"))
	info := res.Info
	assert.NotEmpty(t, info.KeyId)
	assert.Equal(t, "Zapier prod", info.Name)
	assert.Equal(t, res.Key[4:12], info.Prefix)
	assert.Equal(t, "human creation", info.CreatedBy)
	assert.Equal(t, timeRef.Unix(), info.CreatedAt.AsTime().Unix())
	assert.Nil(t, info.ExpiresAt)
	assert.Nil(t, info.LastUsedAt)
	assert.Nil(t, info.RevokedAt)

	// only the hash is kept
	stored := storedKey(t, gormDB, info.KeyId)
	assert.Equal(t, crypt.HashToken(res.Key), stored.KeyHash)
	assert.Equal(t, uint(account), stored.AccountID)
	for _, column := range []string{stored.Name, stored.Prefix, stored.CreatedBy} {
		assert.NotContains(t, column, res.Key)
	}

	// listed, without the key
	listed := listKeys(t, conn, account, realm.Name, false)
	if assert.Len(t, listed.Keys, 1) {
		assert.Equal(t, info.KeyId, listed.Keys[0].KeyId)
		assert.Equal(t, info.Prefix, listed.Keys[0].Prefix)
	}

	// an expiry is kept
	expiring := createKey(t, conn, account, realm.Name, "temporary", timestamppb.New(timeRef.Add(time.Hour)))
	assert.Equal(t, int32(201), expiring.Code)
	assert.Equal(t, timeRef.Add(time.Hour).Unix(), expiring.Info.ExpiresAt.AsTime().Unix())
}

func TestAccessKeyCreateRefusals(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccessKeyCreateRefusals", entities.RealmKindService)
	human := newLoginUser(t, gormDB, "TestAccessKeyCreateRefusals@test.com")
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")
	client := v1.NewAccessKeyClient(conn)
	create := func(req *v1.CreateAccessKeyRequest) *v1.CreateAccessKeyResponse {
		res, err := client.Create(context.Background(), req)
		assert.NoError(t, err)
		return res
	}

	// a person holds no keys
	res := create(&v1.CreateAccessKeyRequest{AccountId: int32(human.ID), Realm: human.RealmName, Name: "one", Actor: "human creation"})
	assert.Equal(t, int32(403), res.Code)
	assert.Contains(t, res.Message, consts.ERR_FORBIDDEN_BY_REALM_KIND)
	// an account is only reached through its realm
	res = create(&v1.CreateAccessKeyRequest{AccountId: account, Realm: human.RealmName, Name: "one", Actor: "human creation"})
	assert.Equal(t, int32(404), res.Code)
	assert.Contains(t, res.Message, consts.ERR_ACCOUNT_NOT_FOUND)
	assert.Equal(t, int32(404), create(&v1.CreateAccessKeyRequest{AccountId: 999999, Realm: realm.Name, Name: "one", Actor: "human creation"}).Code)

	for _, name := range []string{"", strings.Repeat("n", 101)} {
		res = create(&v1.CreateAccessKeyRequest{AccountId: account, Realm: realm.Name, Name: name, Actor: "human creation"})
		assert.Equal(t, int32(422), res.Code)
		assert.Contains(t, res.Message, consts.ERR_INVALID_INPUT_NAME)
	}
	res = create(&v1.CreateAccessKeyRequest{AccountId: account, Realm: realm.Name, Name: "one", Actor: ""})
	assert.Equal(t, int32(422), res.Code)
	assert.Contains(t, res.Message, consts.ERR_INVALID_INPUT_ACTOR)
	res = create(&v1.CreateAccessKeyRequest{AccountId: account, Realm: realm.Name, Name: "one", Actor: "human creation", ExpiresAt: timestamppb.New(timeRef.Add(-time.Second))})
	assert.Equal(t, int32(422), res.Code)
	assert.Contains(t, res.Message, consts.ERR_INVALID_INPUT_EXPIRY)

	assert.Len(t, listKeys(t, conn, account, realm.Name, true).Keys, 0)
}

func TestAccessKeyCapUsesTheLowerOfRealmAndServer(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	layout.AccessKeyMaxActive = 3
	two, five := 2, 5
	inherits := newRealm(t, gormDB, "TestAccessKeyCap-inherits", entities.RealmKindService)
	lower := newRealm(t, gormDB, "TestAccessKeyCap-lower", entities.RealmKindService)
	assert.NoError(t, gormDB.Model(&lower).Update("access_key_max_active", &two).Error)
	higher := newRealm(t, gormDB, "TestAccessKeyCap-higher", entities.RealmKindService)
	assert.NoError(t, gormDB.Model(&higher).Update("access_key_max_active", &five).Error)
	conn := setupRPC(t, layout)
	defer conn.Close()

	for _, trial := range []struct {
		realm entities.Realm
		cap   int
	}{{inherits, 3}, {lower, 2}, {higher, 3}} {
		account := newServiceAccount(t, conn, trial.realm.Name, "sb:org:42")
		for i := 0; i < trial.cap; i++ {
			assert.Equal(t, int32(201), createKey(t, conn, account, trial.realm.Name, "one more", nil).Code, trial.realm.Name)
		}
		res := createKey(t, conn, account, trial.realm.Name, "one too many", nil)
		assert.Equal(t, int32(409), res.Code, trial.realm.Name)
		assert.Contains(t, res.Message, consts.ERR_ACCESS_KEY_LIMIT)
	}

	// revoked and expired keys do not count
	account := newServiceAccount(t, conn, inherits.Name, "sb:org:43")
	keys := make([]*v1.CreateAccessKeyResponse, 3)
	for i := range keys {
		keys[i] = createKey(t, conn, account, inherits.Name, "one more", nil)
	}
	assert.Equal(t, int32(409), createKey(t, conn, account, inherits.Name, "one too many", nil).Code)
	assert.Equal(t, int32(200), revokeKey(t, conn, account, inherits.Name, keys[0].Info.KeyId, "revoked by human").Code)
	assert.Equal(t, int32(201), createKey(t, conn, account, inherits.Name, "in the revoked one's place", nil).Code)
	assert.NoError(t, gormDB.Model(&entities.AccessKey{}).Where("id = ?", keys[1].Info.KeyId).Update("expires_at", timeRef.Add(-time.Second)).Error)
	assert.Equal(t, int32(201), createKey(t, conn, account, inherits.Name, "in the expired one's place", nil).Code)
	assert.Equal(t, int32(409), createKey(t, conn, account, inherits.Name, "one too many", nil).Code)
}

func TestAccessKeyListHidesRevokedAndExpiredUnlessAsked(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccessKeyListHidesRevokedAndExpired", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")

	old := createKey(t, conn, account, realm.Name, "old", nil)
	assert.NoError(t, gormDB.Model(&entities.AccessKey{}).Where("id = ?", old.Info.KeyId).Update("created_at", timeRef.Add(-time.Hour)).Error)
	live := createKey(t, conn, account, realm.Name, "live", nil)
	revoked := createKey(t, conn, account, realm.Name, "revoked", nil)
	assert.Equal(t, int32(200), revokeKey(t, conn, account, realm.Name, revoked.Info.KeyId, "revoked by human").Code)
	expired := createKey(t, conn, account, realm.Name, "expired", timestamppb.New(timeRef.Add(time.Hour)))
	assert.NoError(t, gormDB.Model(&entities.AccessKey{}).Where("id = ?", expired.Info.KeyId).Update("expires_at", timeRef.Add(-time.Second)).Error)

	// live only, newest first
	listed := listKeys(t, conn, account, realm.Name, false).Keys
	if assert.Len(t, listed, 2) {
		assert.Equal(t, live.Info.KeyId, listed[0].KeyId)
		assert.Equal(t, old.Info.KeyId, listed[1].KeyId)
	}

	all := listKeys(t, conn, account, realm.Name, true).Keys
	assert.Len(t, all, 4)
	byID := map[string]*v1.AccessKeyInfo{}
	for _, key := range all {
		byID[key.KeyId] = key
	}
	assert.Equal(t, timeRef.Unix(), byID[revoked.Info.KeyId].RevokedAt.AsTime().Unix())
	assert.Equal(t, "revoked by human", byID[revoked.Info.KeyId].RevokedBy)
	assert.Equal(t, entities.AccessKeyRevokedManual, byID[revoked.Info.KeyId].RevokedReason)
	assert.Nil(t, byID[expired.Info.KeyId].RevokedAt)
	assert.True(t, byID[expired.Info.KeyId].ExpiresAt.AsTime().Before(timeRef))
	assert.Nil(t, byID[live.Info.KeyId].RevokedAt)
}

func TestAccessKeyRevokeRecordsWhoAndRefusesTheRest(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccessKeyRevokeRecordsWho", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")
	other := newServiceAccount(t, conn, realm.Name, "sb:org:43")
	key := createKey(t, conn, account, realm.Name, "one", nil)

	res := revokeKey(t, conn, account, realm.Name, key.Info.KeyId, "")
	assert.Equal(t, int32(422), res.Code)
	assert.Contains(t, res.Message, consts.ERR_INVALID_INPUT_ACTOR)
	// another account's key, or no key at all, is not found
	res = revokeKey(t, conn, other, realm.Name, key.Info.KeyId, "revoked by human")
	assert.Equal(t, int32(404), res.Code)
	assert.Contains(t, res.Message, consts.ERR_ACCESS_KEY_NOT_FOUND)
	assert.Equal(t, int32(404), revokeKey(t, conn, account, realm.Name, "not-a-key-id", "revoked by human").Code)
	assert.False(t, storedKey(t, gormDB, key.Info.KeyId).RevokedAt.Valid)

	assert.Equal(t, int32(200), revokeKey(t, conn, account, realm.Name, key.Info.KeyId, "revoked by human").Code)
	stored := storedKey(t, gormDB, key.Info.KeyId)
	assert.True(t, stored.RevokedAt.Valid)
	assert.Equal(t, timeRef.Unix(), stored.RevokedAt.Time.Unix())
	assert.Equal(t, "revoked by human", *stored.RevokedBy)
	assert.Equal(t, entities.AccessKeyRevokedManual, *stored.RevokedReason)
	// revoked already
	assert.Equal(t, int32(404), revokeKey(t, conn, account, realm.Name, key.Info.KeyId, "revoked by human").Code)

	// an expired key can still be revoked, to record who cleaned up
	expired := createKey(t, conn, account, realm.Name, "expired", timestamppb.New(timeRef.Add(time.Hour)))
	assert.NoError(t, gormDB.Model(&entities.AccessKey{}).Where("id = ?", expired.Info.KeyId).Update("expires_at", timeRef.Add(-time.Second)).Error)
	assert.Equal(t, int32(200), revokeKey(t, conn, account, realm.Name, expired.Info.KeyId, "revoked by human").Code)
}

func TestDeletingTheAccountRevokesItsKeys(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestDeletingTheAccountRevokesItsKeys", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	account := newServiceAccount(t, conn, realm.Name, "sb:org:42")
	first := createKey(t, conn, account, realm.Name, "first", nil)
	second := createKey(t, conn, account, realm.Name, "second", nil)
	// a key revoked before keeps its own record
	assert.Equal(t, int32(200), revokeKey(t, conn, account, realm.Name, second.Info.KeyId, "revoked by human").Code)

	actor := "deleted by human"
	res, err := v1.NewAuthClient(conn).Delete(context.Background(), &v1.AuthIdRequest{Uid: account, Actor: &actor})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	revoked := storedKey(t, gormDB, first.Info.KeyId)
	assert.True(t, revoked.RevokedAt.Valid)
	assert.Equal(t, timeRef.Unix(), revoked.RevokedAt.Time.Unix())
	assert.Equal(t, actor, *revoked.RevokedBy)
	assert.Equal(t, entities.AccessKeyRevokedAccountDeleted, *revoked.RevokedReason)
	before := storedKey(t, gormDB, second.Info.KeyId)
	assert.Equal(t, "revoked by human", *before.RevokedBy)
	assert.Equal(t, entities.AccessKeyRevokedManual, *before.RevokedReason)
	// the account is gone, and so is its key listing
	listed, err := v1.NewAccessKeyClient(conn).List(context.Background(), &v1.ListAccessKeysRequest{AccountId: account, Realm: realm.Name, IncludeRevoked: true})
	assert.NoError(t, err)
	assert.Equal(t, int32(404), listed.Code)
}
