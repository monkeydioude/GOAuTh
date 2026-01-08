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
		Login:   payload.Login,
		Realm:   payload.Realm,
		Data:    payload.Data,
		Against: payload.Against,
	})
	if err != nil {
		return FromErrToResponse(err), nil
	}
	return Ok(), nil
}

// func (h *UserActionRPCHandler) Status(
// 	ctx context.Context,
// 	payload *UserActionRequest,
// ) (*UserActionStatusResponse, error) {
// 	actions, err := services.UserActionStatuses(h.DB, services.UserActionStatusIn{
// 		Login:  payload.Login,
// 		Realm:  payload.Realm,
// 		Action: payload.Action,
// 	})
// 	if err != nil {
// 		resErr := FromErrToResponse(err)
// 		return &UserActionStatusResponse{
// 			Code:    resErr.Code,
// 			Message: resErr.Message,
// 		}, nil
// 	}
// 	return &UserActionStatusResponse{
// 		Code:    200,
// 		Message: "OK",
// 		ActionStatuses: dt.SliceTransform(actions, func(act services.UserActionStatusOut) *UserActionStatus {
// 			var validatedAt *timestamppb.Timestamp = nil
// 			if act.ValidatedAt != nil {
// 				validatedAt = timestamppb.New(act.ValidatedAt.UTC())
// 			}
// 			return &UserActionStatus{
// 				Id:          act.ID,
// 				Action:      act.Action,
// 				CreatedAt:   timestamppb.New(act.CreatedAt.UTC()),
// 				UpdatedAt:   timestamppb.New(act.UpdatedAt.UTC()),
// 				ValidatedAt: validatedAt,
// 			}
// 		}),
// 	}, nil
// }
