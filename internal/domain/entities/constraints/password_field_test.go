package constraints

import (
	"GOAuTh/internal/config/consts"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordConstraint(t *testing.T) {
	passwd := strings.Repeat("a", consts.CONSTRAINT_PASSWORD_MIN_SIZE-1)
	assert.Error(t, PasswordSafetyConstraint(passwd, nil))
	passwd = strings.Repeat("b", consts.CONSTRAINT_PASSWORD_MIN_SIZE)
	assert.NoError(t, PasswordSafetyConstraint(passwd, nil))
	passwd = strings.Repeat("c", consts.CONSTRAINT_PASSWORD_MIN_SIZE)
	old_passwd := strings.Repeat("c", consts.CONSTRAINT_PASSWORD_MIN_SIZE)
	assert.Error(t, PasswordSafetyConstraint(passwd, &old_passwd))
}
