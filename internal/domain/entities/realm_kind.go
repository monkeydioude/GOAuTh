package entities

import (
	stdErr "errors"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/pkg/errors"
)

const (
	// RealmKindHuman holds people, who sign up and log in with a password. It is
	// the kind of every realm from before kinds existed.
	RealmKindHuman string = "human"
	// RealmKindService holds accounts that are not people. They have no
	// password, so no password flow applies to them.
	RealmKindService string = "service"
)

// RealmKind is what a realm's kind lets its users do, one strategy per kind.
// Realm.Strategy picks it.
type RealmKind interface {
	// AssertPasswordFlows refuses the flows that need a password: signup,
	// login, password and login changes, and user actions.
	AssertPasswordFlows() error
}

// HumanKind is the strategy of RealmKindHuman.
type HumanKind struct{}

func (HumanKind) AssertPasswordFlows() error {
	return nil
}

// ServiceKind is the strategy of RealmKindService.
type ServiceKind struct{}

func (ServiceKind) AssertPasswordFlows() error {
	return errors.Forbidden(stdErr.New(consts.ERR_FORBIDDEN_BY_REALM_KIND))
}

// Strategy is the behavior of the realm's kind. An empty kind is human: the
// realm was made before kinds existed, and the column default says so.
func (r Realm) Strategy() RealmKind {
	switch r.Kind {
	case RealmKindService:
		return ServiceKind{}
	default:
		return HumanKind{}
	}
}
