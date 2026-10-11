package functional

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/config/boot"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// rebootForSeedRealms exercises the production boot path without setup's truncation.
func rebootForSeedRealms() (*gorm.DB, error) {
	res := boot.LayoutBoot([]any{entities.User{}, entities.Realm{}, entities.UserAction{}, entities.Session{}, entities.AccessKey{}}, nil, nil)
	if res.IsErr() {
		return nil, res.Error
	}
	return res.Result().DB, nil
}

func closeSeedRealmsDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
}

func setupSeedRealms(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("SEED_REALMS", "")
	_, db, _ := setup()
	closeSeedRealmsDB(t, db)
	// Start with no realms, including setup's default realm.
	require.NoError(t, db.Exec("TRUNCATE TABLE realms CASCADE").Error)
	return db
}

func captureSeedRealmsLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &logs
}

func TestSeedRealmsBootCreatesMissingRealmsAndPreservesSettings(t *testing.T) {
	db := setupSeedRealms(t)
	t.Setenv("SEED_REALMS", "a:human:signup,b:service")
	t.Setenv("ACCESS_KEY_MAX_ACTIVE", "9")
	logs := captureSeedRealmsLogs(t)

	rebooted, err := rebootForSeedRealms()
	require.NoError(t, err)
	closeSeedRealmsDB(t, rebooted)
	var realms []entities.Realm
	require.NoError(t, db.Order("name").Find(&realms).Error)
	require.Len(t, realms, 2)
	assert.Equal(t, "a", realms[0].Name)
	assert.Equal(t, entities.RealmKindHuman, realms[0].Kind)
	assert.True(t, realms[0].AllowNewUser)
	assert.Equal(t, "b", realms[1].Name)
	assert.Equal(t, entities.RealmKindService, realms[1].Kind)
	assert.False(t, realms[1].AllowNewUser)
	for _, realm := range realms {
		assert.Empty(t, realm.Description)
		assert.Nil(t, realm.AccessKeyMaxActive)
		assert.False(t, realm.DeletedAt.Valid)
	}
	assert.Contains(t, logs.String(), `level=INFO msg="created realm from SEED_REALMS" realm=a`)
	assert.Contains(t, logs.String(), `level=INFO msg="created realm from SEED_REALMS" realm=b`)

	// Operator changes survive even if the configured signup flag disagrees.
	require.NoError(t, db.Model(&realms[0]).Update("allow_new_user", false).Error)
	require.NoError(t, db.Model(&realms[1]).Updates(map[string]any{
		"description": "operator description", "access_key_max_active": 3,
	}).Error)
	var before []entities.Realm
	require.NoError(t, db.Order("name").Find(&before).Error)
	logs.Reset()
	again, err := rebootForSeedRealms()
	require.NoError(t, err)
	closeSeedRealmsDB(t, again)
	var after []entities.Realm
	require.NoError(t, db.Order("name").Find(&after).Error)
	assert.Equal(t, before, after)
	assert.NotContains(t, logs.String(), "created realm from SEED_REALMS")
}

func TestSeedRealmsBootRefusesAnExistingDifferentKind(t *testing.T) {
	db := setupSeedRealms(t)
	existing := entities.Realm{Name: "b", Kind: entities.RealmKindHuman}
	require.NoError(t, db.Create(&existing).Error)
	t.Setenv("SEED_REALMS", "b:service")

	_, err := rebootForSeedRealms()

	require.Error(t, err)
	assert.ErrorContains(t, err, `realm "b"`)
	assert.ErrorContains(t, err, `kind "human"`)
	assert.ErrorContains(t, err, `kind is "service"`)
	var stored entities.Realm
	require.NoError(t, db.First(&stored, "name = ?", "b").Error)
	assert.Equal(t, existing.ID, stored.ID)
	assert.Equal(t, entities.RealmKindHuman, stored.Kind)
}

func TestSeedRealmsBootSkipsDeletedRealmsEvenWithADifferentKind(t *testing.T) {
	db := setupSeedRealms(t)
	for _, realm := range []entities.Realm{
		{Name: "deleted-human", Kind: entities.RealmKindHuman},
		{Name: "deleted-service", Kind: entities.RealmKindService},
	} {
		require.NoError(t, db.Create(&realm).Error)
		require.NoError(t, db.Delete(&realm).Error)
	}
	var before []entities.Realm
	require.NoError(t, db.Unscoped().Order("name").Find(&before).Error)
	t.Setenv("SEED_REALMS", "deleted-human:service,deleted-service:service,live:human")
	logs := captureSeedRealmsLogs(t)

	rebooted, err := rebootForSeedRealms()
	require.NoError(t, err)
	closeSeedRealmsDB(t, rebooted)

	var deleted []entities.Realm
	require.NoError(t, db.Unscoped().Where("name IN ?", []string{"deleted-human", "deleted-service"}).Order("name").Find(&deleted).Error)
	assert.Equal(t, before, deleted)
	for _, realm := range deleted {
		assert.Contains(t, logs.String(), `level=WARN msg="skipping soft-deleted realm; remove it from SEED_REALMS" realm=`+realm.Name)
	}
	var live []entities.Realm
	require.NoError(t, db.Find(&live).Error)
	require.Len(t, live, 1)
	assert.Equal(t, "live", live[0].Name)
}

func TestSeedRealmsConcurrentBoots(t *testing.T) {
	db := setupSeedRealms(t)
	t.Setenv("SEED_REALMS", "a:human:signup,b:service")
	type bootResult struct {
		db  *gorm.DB
		err error
	}
	results := make(chan bootResult, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			db, err := rebootForSeedRealms()
			results <- bootResult{db: db, err: err}
		}()
	}
	close(start)
	for range 2 {
		result := <-results
		if assert.NoError(t, result.err) {
			closeSeedRealmsDB(t, result.db)
		}
	}
	var realms []entities.Realm
	require.NoError(t, db.Order("name").Find(&realms).Error)
	require.Len(t, realms, 2)
	assert.Equal(t, "a", realms[0].Name)
	assert.True(t, realms[0].AllowNewUser)
	assert.Equal(t, "b", realms[1].Name)
	assert.False(t, realms[1].AllowNewUser)
}

func TestSeedRealmsEmptyConfigDoesNothing(t *testing.T) {
	db := setupSeedRealms(t)
	rebooted, err := rebootForSeedRealms()
	require.NoError(t, err)
	closeSeedRealmsDB(t, rebooted)
	var count int64
	require.NoError(t, db.Unscoped().Model(&entities.Realm{}).Count(&count).Error)
	assert.Zero(t, count)
}
