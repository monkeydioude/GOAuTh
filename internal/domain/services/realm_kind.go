package services

import (
	stdErr "errors"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities/constraints"
	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"gorm.io/gorm"
)

// actorMaxLength caps the free-form name a consumer gives for who acted.
const actorMaxLength = 255

// RealmKind is what a realm's kind lets its accounts do, one strategy per
// kind. KindOf picks it.
type RealmKind interface {
	// AssertPasswordFlows refuses the flows that need a password: signup,
	// login, password and login changes, and user actions.
	AssertPasswordFlows() error
	// NewAccount is the account a trusted backend makes in the realm for
	// login, on behalf of actor, or the refusal: people sign up instead.
	NewAccount(realm entities.Realm, login string, actor string) (*entities.User, error)
	// Delete closes the account in tx: soft-deleted, what it holds ended.
	// actor is who asked, as the consumer names them.
	Delete(tx *gorm.DB, user *entities.User, actor string, now time.Time) error
	// AssertAccessKeys refuses holding access keys.
	AssertAccessKeys() error
}

// KindOf is the strategy of the realm's kind. An empty kind is human: the
// realm was made before kinds existed, and the column default says so.
func KindOf(realm entities.Realm) RealmKind {
	switch realm.Kind {
	case entities.RealmKindService:
		return ServiceKind{}
	default:
		return HumanKind{}
	}
}

// HumanKind is the strategy of RealmKindHuman: people, with a password.
type HumanKind struct{}

func (HumanKind) AssertPasswordFlows() error {
	return nil
}

// NewAccount refuses: a person signs up with a password.
func (HumanKind) NewAccount(entities.Realm, string, string) (*entities.User, error) {
	return nil, errors.Forbidden(stdErr.New(consts.ERR_FORBIDDEN_BY_REALM_KIND))
}

func (HumanKind) Delete(tx *gorm.DB, user *entities.User, actor string, now time.Time) error {
	return closeAccount(tx, user, actor, now)
}

// AssertAccessKeys refuses: a person logs in instead.
func (HumanKind) AssertAccessKeys() error {
	return errors.Forbidden(stdErr.New(consts.ERR_FORBIDDEN_BY_REALM_KIND))
}

// ServiceKind is the strategy of RealmKindService: accounts that are not
// people, managed by a trusted backend that says who acts.
type ServiceKind struct{}

func (ServiceKind) AssertPasswordFlows() error {
	return errors.Forbidden(stdErr.New(consts.ERR_FORBIDDEN_BY_REALM_KIND))
}

// NewAccount is an account without a password, with a slug for a login.
func (ServiceKind) NewAccount(realm entities.Realm, login string, actor string) (*entities.User, error) {
	if err := constraints.SlugConstraint(login, nil); err != nil {
		return nil, errors.UnprocessableEntity(err)
	}
	if err := assertActor(actor); err != nil {
		return nil, err
	}
	return &entities.User{
		Login:     login,
		Password:  "",
		RealmID:   realm.ID,
		RealmName: realm.Name,
		CreatedBy: &actor,
	}, nil
}

func (ServiceKind) Delete(tx *gorm.DB, user *entities.User, actor string, now time.Time) error {
	if err := assertActor(actor); err != nil {
		return err
	}
	return closeAccount(tx, user, actor, now)
}

func (ServiceKind) AssertAccessKeys() error {
	return nil
}

// assertActor checks the free-form name of who acts, which a consumer must give.
func assertActor(actor string) error {
	if actor == "" || len(actor) > actorMaxLength {
		return errors.UnprocessableEntity(stdErr.New(consts.ERR_INVALID_INPUT_ACTOR))
	}
	return nil
}

// closeAccount soft-deletes the account and revokes its sessions and access
// keys, whatever its kind: nothing it holds survives it.
func closeAccount(tx *gorm.DB, user *entities.User, actor string, now time.Time) error {
	if err := tx.Delete(&entities.User{}, user.ID).Error; err != nil {
		return err
	}
	if err := revokeUserSessions(tx, user.ID, "", entities.SessionRevokedAccountDeactivated, now); err != nil {
		return err
	}
	return revokeAccountKeys(tx, user.ID, actor, now)
}
