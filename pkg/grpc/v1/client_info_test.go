package v1

import (
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/domain/services"

	"github.com/stretchr/testify/assert"
)

func TestIntoClientInfo(t *testing.T) {
	// a caller that sends no client
	assert.Equal(t, services.ClientInfo{}, (&UserRequest{}).GetClient().IntoClientInfo())
	assert.Equal(t, services.ClientInfo{}, (&RefreshIn{}).GetClient().IntoClientInfo())

	trial := &RefreshIn{Client: &ClientInfo{Ip: "::ffff:203.0.113.7", UserAgent: "Mozilla/5.0"}}
	assert.Equal(t, services.ClientInfo{IP: "203.0.113.7", UserAgent: "Mozilla/5.0"}, trial.GetClient().IntoClientInfo())
	assert.Equal(t, "", (&ClientInfo{Ip: "garbage"}).IntoClientInfo().IP)
}
