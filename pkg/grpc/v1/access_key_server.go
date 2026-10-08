package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/calqs/gopkg/dt"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// AccessKeyRPCHandler serves the access keys of the accounts a trusted backend
// manages.
type AccessKeyRPCHandler struct {
	UnimplementedAccessKeyServer
	DB                 *gorm.DB
	AccessTokenFactory *services.JWTFactory
	// MaxActive is the live keys an account may hold; a realm may cap lower
	MaxActive int
}

func NewAccessKeyRPCHandler(layout *handlers.Layout) *AccessKeyRPCHandler {
	return &AccessKeyRPCHandler{
		DB:                 layout.DB,
		AccessTokenFactory: layout.AccessTokenFactory,
		MaxActive:          layout.AccessKeyMaxActive,
	}
}

func (h *AccessKeyRPCHandler) Create(ctx context.Context, req *CreateAccessKeyRequest) (*CreateAccessKeyResponse, error) {
	if req == nil {
		return &CreateAccessKeyResponse{Code: http.StatusInternalServerError, Message: "no req pointer"}, nil
	}
	var expiresAt *time.Time
	if req.GetExpiresAt() != nil {
		at := req.GetExpiresAt().AsTime()
		expiresAt = &at
	}
	key, info, err := services.AccessKeyCreate(h.DB, services.AccessKeyCreateIn{
		AccountID: uint(req.GetAccountId()),
		Realm:     req.GetRealm(),
		Name:      req.GetName(),
		ExpiresAt: expiresAt,
		Actor:     req.GetActor(),
	}, h.MaxActive, h.AccessTokenFactory.TimeFn())
	if err != nil {
		res := FromErrToResponse(err)
		return &CreateAccessKeyResponse{Code: res.Code, Message: res.Message}, nil
	}
	return &CreateAccessKeyResponse{
		Code:    http.StatusCreated,
		Message: "Created",
		Key:     key,
		Info:    intoAccessKeyInfo(*info),
	}, nil
}

func (h *AccessKeyRPCHandler) List(ctx context.Context, req *ListAccessKeysRequest) (*ListAccessKeysResponse, error) {
	if req == nil {
		return &ListAccessKeysResponse{Code: http.StatusInternalServerError, Message: "no req pointer"}, nil
	}
	keys, err := services.AccessKeyList(h.DB, uint(req.GetAccountId()), req.GetRealm(), req.GetIncludeRevoked(), h.AccessTokenFactory.TimeFn())
	if err != nil {
		res := FromErrToResponse(err)
		return &ListAccessKeysResponse{Code: res.Code, Message: res.Message}, nil
	}
	return &ListAccessKeysResponse{
		Code:    http.StatusOK,
		Message: "Ok",
		Keys:    dt.SliceTransform(keys, intoAccessKeyInfo),
	}, nil
}

func (h *AccessKeyRPCHandler) Revoke(ctx context.Context, req *RevokeAccessKeyRequest) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), nil
	}
	if err := services.AccessKeyRevoke(h.DB, uint(req.GetAccountId()), req.GetRealm(), req.GetKeyId(), req.GetActor(), h.AccessTokenFactory.TimeFn()); err != nil {
		return FromErrToResponse(err), nil
	}
	return Ok(), nil
}

// intoAccessKeyInfo shapes a key for a consumer: everything but the hash.
func intoAccessKeyInfo(key entities.AccessKey) *AccessKeyInfo {
	info := &AccessKeyInfo{
		KeyId:     key.ID.String(),
		Name:      key.Name,
		Prefix:    key.Prefix,
		CreatedAt: timestamppb.New(key.CreatedAt.UTC()),
		CreatedBy: key.CreatedBy,
	}
	if key.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(key.ExpiresAt.UTC())
	}
	if key.LastUsedAt != nil {
		info.LastUsedAt = timestamppb.New(key.LastUsedAt.UTC())
	}
	if key.RevokedAt.Valid {
		info.RevokedAt = timestamppb.New(key.RevokedAt.Time.UTC())
	}
	if key.RevokedBy != nil {
		info.RevokedBy = *key.RevokedBy
	}
	if key.RevokedReason != nil {
		info.RevokedReason = *key.RevokedReason
	}
	return info
}
