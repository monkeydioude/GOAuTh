package v1

import "github.com/monkeydioude/goauth/v2/internal/domain/services"

// IntoClientInfo maps the client a gRPC caller sent on behalf of the end user.
// A missing client gives empty values.
func (c *ClientInfo) IntoClientInfo() services.ClientInfo {
	return services.NewClientInfo(c.GetIp(), c.GetUserAgent())
}
