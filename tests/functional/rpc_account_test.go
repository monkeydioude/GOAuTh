package functional

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// newRealm creates a realm of kind, deleted with its users at the end of the test.
func newRealm(t *testing.T, gormDB *gorm.DB, name string, kind string) entities.Realm {
	realm := entities.Realm{ID: uuid.New(), Name: name, AllowNewUser: true, Kind: kind}
	assert.NoError(t, gormDB.Create(&realm).Error)
	t.Cleanup(func() {
		gormDB.Unscoped().Delete(&entities.User{}, "realm_id = ?", realm.ID)
		gormDB.Unscoped().Delete(&realm)
	})
	return realm
}

func TestAccountCreateMakesAServiceAccountThatCannotLogIn(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccountCreateMakesAServiceAccountThatCannotLogIn", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	ctx := context.Background()

	res, err := v1.NewAccountClient(conn).Create(ctx, &v1.CreateAccountRequest{Realm: realm.Name, Login: "sb:org:42", Actor: "human creation"})
	assert.NoError(t, err)
	assert.Equal(t, int32(201), res.Code)
	assert.NotZero(t, res.AccountId)
	assert.Equal(t, "sb:org:42", res.Login)
	assert.Equal(t, realm.Name, res.Realm)
	assert.Equal(t, entities.RealmKindService, res.RealmKind)
	assert.NotNil(t, res.CreatedAt)

	stored := storedUser(t, gormDB, uint(res.AccountId))
	assert.Equal(t, "", stored.Password)
	assert.Equal(t, realm.ID, stored.RealmID)
	if assert.NotNil(t, stored.CreatedBy) {
		assert.Equal(t, "human creation", *stored.CreatedBy)
	}

	// no password, no login: neither with the stored value nor with another
	for _, password := range []string{"", "test"} {
		login, err := v1.NewAuthClient(conn).Login(ctx, &v1.UserRequest{Login: "sb:org:42", Password: password, Realm: realm.Name})
		assert.NoError(t, err)
		assert.Equal(t, int32(401), login.Code)
	}
}

func TestAccountCreateRefusesHumanRealms(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccountCreateRefusesHumanRealms", entities.RealmKindHuman)
	conn := setupRPC(t, layout)
	defer conn.Close()

	res, err := v1.NewAccountClient(conn).Create(context.Background(), &v1.CreateAccountRequest{Realm: realm.Name, Login: "bots", Actor: "human creation"})
	assert.NoError(t, err)
	assert.Equal(t, int32(403), res.Code)
	assert.Contains(t, res.Message, consts.ERR_FORBIDDEN_BY_REALM_KIND)
	var count int64
	assert.NoError(t, gormDB.Model(&entities.User{}).Where("realm_id = ?", realm.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestAccountCreateRefusesBadInput(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccountCreateRefusesBadInput", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewAccountClient(conn)
	create := func(realmName string, login string, actor string) *v1.CreateAccountResponse {
		res, err := client.Create(context.Background(), &v1.CreateAccountRequest{Realm: realmName, Login: login, Actor: actor})
		assert.NoError(t, err)
		return res
	}

	assert.Equal(t, int32(404), create("no-such-realm", "bots", "human creation").Code)
	// never an email, never uppercase, never empty
	for _, login := range []string{"bots@example.com", "Bots", ""} {
		res := create(realm.Name, login, "human creation")
		assert.Equal(t, int32(422), res.Code, login)
		assert.Contains(t, res.Message, consts.ERR_INVALID_INPUT_LOGIN)
	}
	for _, actor := range []string{"", strings.Repeat("a", 256)} {
		res := create(realm.Name, "bots", actor)
		assert.Equal(t, int32(422), res.Code)
		assert.Contains(t, res.Message, consts.ERR_INVALID_INPUT_ACTOR)
	}
	assert.Equal(t, int32(201), create(realm.Name, "bots", "human creation").Code)
	// a login the realm has: that account again (TestAccountCreateIsIdempotent)
	assert.Equal(t, int32(200), create(realm.Name, "bots", "human creation").Code)
	// a slug is free in another realm
	other := newRealm(t, gormDB, "TestAccountCreateRefusesBadInput-other", entities.RealmKindService)
	assert.Equal(t, int32(201), create(other.Name, "bots", "human creation").Code)

	var count int64
	assert.NoError(t, gormDB.Model(&entities.User{}).Where("login = ?", "bots").Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestAccountCreateIsIdempotent(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccountCreateIsIdempotent", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewAccountClient(conn)
	ctx := context.Background()

	first, err := client.Create(ctx, &v1.CreateAccountRequest{Realm: realm.Name, Login: "sb:org:42", Actor: "human creation"})
	assert.NoError(t, err)
	assert.Equal(t, int32(201), first.Code)
	// the consumer lost the id: it asks again, maybe as someone else
	again, err := client.Create(ctx, &v1.CreateAccountRequest{Realm: realm.Name, Login: "sb:org:42", Actor: "retry job"})
	assert.NoError(t, err)

	assert.Equal(t, int32(200), again.Code)
	assert.Equal(t, first.AccountId, again.AccountId)
	assert.Equal(t, "sb:org:42", again.Login)
	assert.Equal(t, realm.Name, again.Realm)
	assert.Equal(t, entities.RealmKindService, again.RealmKind)
	assert.True(t, first.CreatedAt.AsTime().Equal(again.CreatedAt.AsTime()))
	stored := storedUser(t, gormDB, uint(again.AccountId))
	if assert.NotNil(t, stored.CreatedBy) {
		assert.Equal(t, "human creation", *stored.CreatedBy, "the account as it was made")
	}
}

func TestAccountCreateAtOnceMakesOneAccount(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAccountCreateAtOnceMakesOneAccount", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewAccountClient(conn)

	answers := make(chan *v1.CreateAccountResponse, 10)
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := client.Create(context.Background(), &v1.CreateAccountRequest{Realm: realm.Name, Login: "sb:org:42", Actor: "human creation"})
			assert.NoError(t, err)
			answers <- res
		}()
	}
	wg.Wait()
	close(answers)

	codes := map[int32]int{}
	ids := map[int32]bool{}
	for res := range answers {
		codes[res.Code]++
		ids[res.AccountId] = true
	}
	assert.Equal(t, map[int32]int{201: 1, 200: 9}, codes)
	assert.Len(t, ids, 1, "every answer names the same account")
	var count int64
	assert.NoError(t, gormDB.Model(&entities.User{}).Where("realm_id = ?", realm.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestAuthDeleteNeedsAnActorForServiceAccounts(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	realm := newRealm(t, gormDB, "TestAuthDeleteNeedsAnActorForServiceAccounts", entities.RealmKindService)
	conn := setupRPC(t, layout)
	defer conn.Close()
	ctx := context.Background()
	accounts := v1.NewAccountClient(conn)
	created, err := accounts.Create(ctx, &v1.CreateAccountRequest{Realm: realm.Name, Login: "sb:org:42", Actor: "human creation"})
	assert.NoError(t, err)
	assert.Equal(t, int32(201), created.Code)
	auth := v1.NewAuthClient(conn)

	res, err := auth.Delete(ctx, &v1.AuthIdRequest{Uid: created.AccountId})
	assert.NoError(t, err)
	assert.Equal(t, int32(422), res.Code)
	assert.Contains(t, res.Message, consts.ERR_INVALID_INPUT_ACTOR)
	assert.NoError(t, gormDB.First(&entities.User{}, created.AccountId).Error)

	actor := "revoked by human"
	res, err = auth.Delete(ctx, &v1.AuthIdRequest{Uid: created.AccountId, Actor: &actor})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	var gone entities.User
	assert.NoError(t, gormDB.Unscoped().First(&gone, created.AccountId).Error)
	assert.True(t, gone.DeletedAt.Valid)
	// gone already: a no-op
	res, err = auth.Delete(ctx, &v1.AuthIdRequest{Uid: created.AccountId, Actor: &actor})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	// the login is free again
	again, err := accounts.Create(ctx, &v1.CreateAccountRequest{Realm: realm.Name, Login: "sb:org:42", Actor: "human creation"})
	assert.NoError(t, err)
	assert.Equal(t, int32(201), again.Code)
	assert.NotEqual(t, created.AccountId, again.AccountId)
}
