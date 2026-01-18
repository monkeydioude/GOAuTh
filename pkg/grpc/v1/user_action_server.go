package v1

import (
	"context"

	"github.com/google/uuid"
	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/domain/models"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"gorm.io/gorm"
)

type UserActionRPCHandler struct {
	UnimplementedUserActionServer
	DB         *gorm.DB
	UserParams *models.UsersParams
}

func NewUserActionRPCHandler(layout *handlers.Layout) *UserActionRPCHandler {
	return &UserActionRPCHandler{
		DB:         layout.DB,
		UserParams: layout.UserParams,
	}
}

func (h *UserActionRPCHandler) Create(
	ctx context.Context,
	payload *UserActionRequest,
) (*Response, error) {
	res, err := services.UserActionCreate(h.DB, services.UserActionCreateIn{
		Login:  payload.Login,
		Realm:  payload.Realm,
		Action: payload.Action,
	},
		func() string { return uuid.NewString() },
	)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	return Created(res.Data), nil
}

func (h *UserActionRPCHandler) Validate(
	ctx context.Context,
	payload *UserActionValidation,
) (*Response, error) {
	err := services.UserActionValidate(h.DB, h.UserParams, services.UserActionValidateIn{
		Realm:   payload.Realm,
		Data:    payload.Data,
		Against: payload.Against,
	})
	if err != nil {
		return FromErrToResponse(err), nil
	}
	return Ok(), nil
}
