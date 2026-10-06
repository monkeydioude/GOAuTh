package boot

import (
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"

	"github.com/stretchr/testify/assert"
)

func TestLayoutBootRefusesInvalidTrustedProxies(t *testing.T) {
	t.Setenv(consts.TRUSTED_PROXIES, "10.0.0.0/8, not-a-cidr")

	res := LayoutBoot(nil, nil, nil)

	assert.True(t, res.IsErr())
}
