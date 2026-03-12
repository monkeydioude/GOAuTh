package v1

import (
	"context"

	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/http/rpc"
	"google.golang.org/grpc/metadata"
)

func FetchAccessToken(ctx context.Context, statusIn *StatusIn) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", StatusBadRequest("didnt find any metadata")
	}
	cookie, err := rpc.FetchCookie(md, consts.AuthorizationCookie)
	if err != nil {
		if statusIn.AccessToken != "" {
			return statusIn.AccessToken, nil
		}
		return "", err
	}
	return services.GetTokenFromBearer(cookie.Value)
}
