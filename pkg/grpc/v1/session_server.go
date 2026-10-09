package v1

import (
	"context"
	stdErr "errors"
	"net/http"

	"github.com/calqs/gopkg/dt"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/errors"
	"github.com/monkeydioude/goauth/v2/pkg/http/rpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// SessionRPCHandler serves the sessions of the user whose access token is in
// the Authorization metadata.
type SessionRPCHandler struct {
	UnimplementedSessionServer
	DB                 *gorm.DB
	AccessTokenFactory *services.JWTFactory
}

// caller reads the claims of the access token the call was made with.
// The token must be valid and its session active.
func (h *SessionRPCHandler) caller(ctx context.Context) (crypt.JWTDefaultClaims, error) {
	cookie, err := rpc.FetchCookieFromContext(ctx, consts.AuthorizationCookie)
	if err != nil {
		return crypt.JWTDefaultClaims{}, errors.Unauthorized(err)
	}
	token, err := services.GetTokenFromBearer(cookie.Value)
	if err != nil {
		return crypt.JWTDefaultClaims{}, err
	}
	jwt, err := services.AuthenticateAccessToken(ctx, token, *h.AccessTokenFactory)
	if err != nil {
		return crypt.JWTDefaultClaims{}, err
	}
	return jwt.Claims, nil
}

func (h *SessionRPCHandler) List(ctx context.Context, req *ListSessionsRequest) (*ListSessionsResponse, error) {
	claims, err := h.caller(ctx)
	if err != nil {
		return fromErrToListSessionsResponse(err), nil
	}
	sessions, err := services.ListSessions(h.DB.WithContext(ctx), claims.UID, req.GetIncludeRevoked(), h.AccessTokenFactory.TimeFn())
	if err != nil {
		return fromErrToListSessionsResponse(errors.DBError(err)), nil
	}
	return &ListSessionsResponse{
		Code:    http.StatusOK,
		Message: "Ok",
		Sessions: dt.SliceTransform(sessions, func(session entities.Session) *SessionInfo {
			return intoSessionInfo(session, claims.SID)
		}),
	}, nil
}

func (h *SessionRPCHandler) Revoke(ctx context.Context, req *RevokeSessionRequest) (*Response, error) {
	claims, err := h.caller(ctx)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	revoked, err := services.RevokeSession(h.DB.WithContext(ctx), claims.UID, req.GetSessionId(), entities.SessionRevokedByUser, h.AccessTokenFactory.TimeFn())
	if err != nil {
		return FromErrToResponse(errors.DBError(err)), nil
	}
	// another user's session is not found either
	if !revoked {
		return FromErrToResponse(errors.NotFound(stdErr.New(consts.ERR_SESSION_NOT_FOUND))), nil
	}
	return Ok(), nil
}

func (h *SessionRPCHandler) RevokeAll(ctx context.Context, req *RevokeAllSessionsRequest) (*Response, error) {
	claims, err := h.caller(ctx)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	keepSID := ""
	if req.GetKeepCurrent() {
		keepSID = claims.SID
	}
	if err := services.RevokeAllSessions(h.DB.WithContext(ctx), claims.UID, keepSID, h.AccessTokenFactory.TimeFn()); err != nil {
		return FromErrToResponse(errors.DBError(err)), nil
	}
	return Ok(), nil
}

func intoSessionInfo(session entities.Session, currentSID string) *SessionInfo {
	info := &SessionInfo{
		Id:             session.ID.String(),
		UserAgent:      session.UserAgent,
		LoginIp:        session.LoginIP,
		LastIp:         session.LastIP,
		CreatedAt:      timestamppb.New(session.CreatedAt.UTC()),
		LastConnection: timestamppb.New(session.LastConnection.UTC()),
		ExpiresAt:      timestamppb.New(session.ExpiresAt.UTC()),
		Current:        session.ID.String() == currentSID,
	}
	if session.DeletedAt.Valid {
		info.RevokedAt = timestamppb.New(session.DeletedAt.Time.UTC())
	}
	if session.RevokedReason != nil {
		info.RevokedReason = *session.RevokedReason
	}
	return info
}

func fromErrToListSessionsResponse(err error) *ListSessionsResponse {
	res := FromErrToResponse(err)
	return &ListSessionsResponse{
		Code:    res.Code,
		Message: res.Message,
	}
}

func NewSessionRPCHandler(layout *handlers.Layout) *SessionRPCHandler {
	return &SessionRPCHandler{
		DB:                 layout.DB,
		AccessTokenFactory: layout.AccessTokenFactory,
	}
}
