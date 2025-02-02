package slice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestICanMapVarsToSliceElements(t *testing.T) {
	t.Run("test 1", func(t *testing.T) {
		trial := []string{"salut", "les", "kids"}
		var salut, les, kids string

		MapVars(trial, &salut, &les, &kids)
		assert.Equal(t, trial[0], salut)
		assert.Equal(t, trial[1], les)
		assert.Equal(t, trial[2], kids)
	})
}
