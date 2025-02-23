package ptr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestICanMakeAPtrOrNilOnEmptyValue(t *testing.T) {
	assert.Nil(t, PtrNilOnEmpty(""))
	assert.Equal(t, "a", *PtrNilOnEmpty("a"))
}
