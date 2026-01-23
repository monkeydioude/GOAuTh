package v1

import (
	"context"
	"net/http"

	"github.com/calqs/gopkg/dt"
	"github.com/google/uuid"
	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/domain/models"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		Data:   dt.PtrNilOnEmpty(payload.Data),
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
	email, err := services.UserActionValidate(h.DB, h.UserParams, services.UserActionValidateIn{
		Realm:   payload.Realm,
		Data:    payload.Data,
		Against: payload.Against,
	})
	if err != nil {
		return FromErrToResponse(err), nil
	}
	return &Response{
		Code:       http.StatusOK,
		Identifier: email,
		Message:    "Ok",
	}, nil
}

func (h *UserActionRPCHandler) Status(
	ctx context.Context,
	payload *UserActionRequest,
) (*UserActionStatusResponse, error) {
	res, err := services.UserActionStatus(h.DB, services.UserActionStatusIn{
		Realm:  payload.Realm,
		Login:  payload.Login,
		Action: payload.Action,
	})
	if err != nil {
		return FromErrToUserActionStatusResponse(err), nil
	}
	return &UserActionStatusResponse{
		Code:    http.StatusOK,
		Message: "Ok",
		Statuses: dt.SliceTransform(res, func(uas services.UserActionStatusOut) *UserActionStatus {
			res := &UserActionStatus{
				Realm:     uas.Realm,
				Login:     uas.Login,
				Action:    uas.Action,
				Data:      uas.Data,
				CreatedAt: timestamppb.New(uas.CreatedAt.UTC()),
				UpdatedAt: timestamppb.New(uas.UpdatedAt.UTC()),
			}
			if uas.ValidatedAt != nil {
				res.ValidatedAt = timestamppb.New(*uas.ValidatedAt)
			}
			return res
		}),
	}, nil
}
