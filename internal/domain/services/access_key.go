package services

import (
	stdErr "errors"
	"log/slog"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/config/logs"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AccessKeyCreateIn struct {
	AccountID uint
	Realm     string
	Name      string
	// nil never expires
	ExpiresAt *time.Time
	// who asked for it, as the consumer names them
	Actor string
}

// accountInRealm loads the account and its realm. The account must belong to
// the named realm: a uid from another realm is "not found", same as an unknown
// one, so a mixed-up uid and realm pair can't act on the wrong account.
func accountInRealm(db *gorm.DB, accountID uint, realmName string) (entities.User, entities.Realm, error) {
	var realm entities.Realm
	if err := db.Where("name = ?", realmName).First(&realm).Error; err != nil {
		if stdErr.Is(err, gorm.ErrRecordNotFound) {
			return entities.User{}, entities.Realm{}, errors.NotFound(stdErr.New(consts.ERR_ACCOUNT_NOT_FOUND))
		}
		return entities.User{}, entities.Realm{}, errors.DBError(err)
	}
	var account entities.User
	if err := db.First(&account, "id = ? AND realm_id = ?", accountID, realm.ID).Error; err != nil {
		if stdErr.Is(err, gorm.ErrRecordNotFound) {
			return entities.User{}, entities.Realm{}, errors.NotFound(stdErr.New(consts.ERR_ACCOUNT_NOT_FOUND))
		}
		return entities.User{}, entities.Realm{}, errors.DBError(err)
	}
	return account, realm, nil
}

// effectiveAccessKeyCap is how many live keys an account of the realm may
// hold: the realm's own cap when it has one, never above the server's.
func effectiveAccessKeyCap(realm entities.Realm, serverCap int) int {
	if realm.AccessKeyMaxActive != nil && *realm.AccessKeyMaxActive < serverCap {
		return *realm.AccessKeyMaxActive
	}
	return serverCap
}

// liveAccessKeys narrows to keys not expired; the soft-delete scope already
// leaves out the revoked ones.
func liveAccessKeys(query *gorm.DB, now time.Time) *gorm.DB {
	return query.Where("(expires_at IS NULL OR expires_at > ?)", now)
}

// AccessKeyCreate mints a key for the account and hands it back in clear, the
// only time it ever is: only its hash gets kept. serverCap is how many live
// keys an account may hold; the realm may cap lower.
func AccessKeyCreate(db *gorm.DB, in AccessKeyCreateIn, serverCap int, now time.Time) (string, *entities.AccessKey, error) {
	if db == nil {
		return "", nil, errors.InternalServerError(stdErr.New("nil db object"))
	}
	account, realm, err := accountInRealm(db, in.AccountID, in.Realm)
	if err != nil {
		return "", nil, err
	}
	if err := KindOf(realm).AssertAccessKeys(); err != nil {
		return "", nil, err
	}
	if in.Name == "" || len(in.Name) > entities.AccessKeyNameMaxLength {
		return "", nil, errors.UnprocessableEntity(stdErr.New(consts.ERR_INVALID_INPUT_NAME))
	}
	if err := assertActor(in.Actor); err != nil {
		return "", nil, err
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(now) {
		return "", nil, errors.UnprocessableEntity(stdErr.New(consts.ERR_INVALID_INPUT_EXPIRY))
	}
	key, err := crypt.NewAccessKey()
	if err != nil {
		return "", nil, errors.InternalServerError(err)
	}
	row := &entities.AccessKey{
		AccountID: account.ID,
		Name:      in.Name,
		KeyHash:   crypt.HashToken(key),
		Prefix:    crypt.AccessKeyPrefixOf(key),
		CreatedAt: now,
		CreatedBy: in.Actor,
		ExpiresAt: in.ExpiresAt,
	}
	limit := effectiveAccessKeyCap(realm, serverCap)
	err = db.Transaction(func(tx *gorm.DB) error {
		// the account row is locked: two creations racing cannot both pass the cap
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&entities.User{}, account.ID).Error; err != nil {
			return err
		}
		var live int64
		if err := liveAccessKeys(tx.Model(&entities.AccessKey{}).Where("account_id = ?", account.ID), now).Count(&live).Error; err != nil {
			return err
		}
		if int(live) >= limit {
			return errors.Conflict(stdErr.New(consts.ERR_ACCESS_KEY_LIMIT))
		}
		return tx.Create(row).Error
	})
	if err != nil {
		if _, typed := err.(errors.Err); typed {
			return "", nil, err
		}
		return "", nil, errors.DBError(err)
	}
	return key, row, nil
}

// AccessKeyList lists the account's live keys, newest first. includeRevoked
// adds the revoked and expired ones.
func AccessKeyList(db *gorm.DB, accountID uint, realmName string, includeRevoked bool, now time.Time) ([]entities.AccessKey, error) {
	account, _, err := accountInRealm(db, accountID, realmName)
	if err != nil {
		return nil, err
	}
	query := db.Where("account_id = ?", account.ID)
	if includeRevoked {
		query = query.Unscoped()
	} else {
		query = liveAccessKeys(query, now)
	}
	var keys []entities.AccessKey
	if err := query.Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, errors.DBError(err)
	}
	return keys, nil
}

// AccessKeyRevoke ends one key of the account by hand, recording who asked. A
// key expired but not revoked yet can still be; one already revoked, or of
// another account, is not found.
func AccessKeyRevoke(db *gorm.DB, accountID uint, realmName string, keyID string, actor string, now time.Time) error {
	account, _, err := accountInRealm(db, accountID, realmName)
	if err != nil {
		return err
	}
	if err := assertActor(actor); err != nil {
		return err
	}
	id, err := uuid.Parse(keyID)
	if err != nil {
		return errors.NotFound(stdErr.New(consts.ERR_ACCESS_KEY_NOT_FOUND))
	}
	res := revokeAccessKeys(db, actor, entities.AccessKeyRevokedManual, now, "id = ? AND account_id = ?", id, account.ID)
	if res.Error != nil {
		return errors.DBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NotFound(stdErr.New(consts.ERR_ACCESS_KEY_NOT_FOUND))
	}
	return nil
}

// accessKeyUsedWindow is how long a key's last_used_at is left alone: it says
// "in use lately", not when exactly, and that spares a write per verification.
const accessKeyUsedWindow = time.Minute

// VerifiedAccessKey is whose a live key is.
type VerifiedAccessKey struct {
	KeyID     uuid.UUID
	ExpiresAt *time.Time
	AccountID uint
	Login     string
	Realm     string
	RealmKind string
}

// AccessKeyVerify tells whose key is. It is valid while it is not revoked or
// expired and its account, which belongs to a realm still there, is not deleted
// or revoked: one query, and every other case is the same errors.Unauthorized
// InvalidKey, so a dead key learns nothing about why. A DB failure is not the
// key's fault, and says so. There is no cache: a revocation shows at once.
func AccessKeyVerify(db *gorm.DB, key string, now time.Time) (*VerifiedAccessKey, error) {
	if db == nil {
		return nil, errors.InternalServerError(stdErr.New("nil db object"))
	}
	if !crypt.IsAccessKey(key) {
		return nil, errors.Unauthorized(stdErr.New(consts.ERR_INVALID_KEY))
	}
	var verified VerifiedAccessKey
	// the soft-delete scope leaves out the revoked keys
	res := db.Model(&entities.AccessKey{}).
		Select("access_keys.id AS key_id, access_keys.expires_at, users.id AS account_id, users.login, realms.name AS realm, realms.kind AS realm_kind").
		Joins("JOIN users ON users.id = access_keys.account_id AND users.deleted_at IS NULL AND (users.revoked_at IS NULL OR users.revoked_at > ?)", now).
		Joins("JOIN realms ON realms.id = users.realm_id AND realms.deleted_at IS NULL").
		Where("access_keys.key_hash = ? AND (access_keys.expires_at IS NULL OR access_keys.expires_at > ?)", crypt.HashToken(key), now).
		Limit(1).
		Scan(&verified)
	if res.Error != nil {
		return nil, errors.DBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, errors.Unauthorized(stdErr.New(consts.ERR_INVALID_KEY))
	}
	touchAccessKey(db, verified.KeyID, now)
	return &verified, nil
}

// touchAccessKey writes last_used_at, at most once per accessKeyUsedWindow. A
// failed write is logged and nothing more: the key was verified.
func touchAccessKey(db *gorm.DB, keyID uuid.UUID, now time.Time) {
	err := db.Model(&entities.AccessKey{}).
		Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", keyID, now.Add(-accessKeyUsedWindow)).
		Update("last_used_at", now).Error
	if err != nil {
		slog.WarnContext(logs.DBContext(db), "could not write the access key's last use", "key_id", keyID, "error", err.Error())
	}
}

// revokeAccessKeys revokes the live keys the query matches: the soft-delete
// scope leaves the revoked ones as they are.
func revokeAccessKeys(tx *gorm.DB, actor string, reason string, now time.Time, query string, args ...any) *gorm.DB {
	values := map[string]any{"revoked_at": now, "revoked_reason": reason}
	if actor != "" {
		values["revoked_by"] = actor
	}
	return tx.Model(&entities.AccessKey{}).Where(query, args...).Updates(values)
}

// revokeAccountKeys revokes every live key of the account, as part of closing it.
func revokeAccountKeys(tx *gorm.DB, accountID uint, actor string, now time.Time) error {
	return revokeAccessKeys(tx, actor, entities.AccessKeyRevokedAccountDeleted, now, "account_id = ?", accountID).Error
}
