package v1

import (
	"context"
	"net/http"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// AccountRPCHandler serves the accounts a trusted backend manages: those of
// service realms. People sign up through Auth instead.
type AccountRPCHandler struct {
	UnimplementedAccountServer
	DB *gorm.DB
}

func NewAccountRPCHandler(layout *handlers.Layout) *AccountRPCHandler {
	return &AccountRPCHandler{
		DB: layout.DB,
	}
}

func fromErrToCreateAccountResponse(err error) *CreateAccountResponse {
	res := FromErrToResponse(err)
	return &CreateAccountResponse{
		Code:    res.Code,
		Message: res.Message,
	}
}

// Create makes an account in realm for login, on behalf of actor, the way the
// realm's kind says: 201. A login the realm's live accounts already have gives
// that account back: 200.
func (h *AccountRPCHandler) Create(ctx context.Context, req *CreateAccountRequest) (*CreateAccountResponse, error) {
	if req == nil {
		return &CreateAccountResponse{Code: http.StatusInternalServerError, Message: "no req pointer"}, nil
	}
	account, created, err := services.AccountCreate(h.DB.WithContext(ctx), services.AccountCreateIn{
		Realm: req.GetRealm(),
		Login: req.GetLogin(),
		Actor: req.GetActor(),
	})
	if err != nil {
		return fromErrToCreateAccountResponse(err), nil
	}
	code, message := http.StatusOK, "Ok"
	if created {
		code, message = http.StatusCreated, "Created"
	}
	return &CreateAccountResponse{
		Code:      int32(code),
		Message:   message,
		AccountId: int32(account.ID),
		Login:     account.Login,
		Realm:     account.Realm.Name,
		RealmKind: account.Realm.Kind,
		CreatedAt: timestamppb.New(account.CreatedAt),
	}, nil
}
