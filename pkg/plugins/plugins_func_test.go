package plugins_test

import (
	"testing"

	"github.com/monkeydioude/goauth/pkg/data_types/tuple"
	"github.com/monkeydioude/goauth/pkg/plugins"

	"github.com/stretchr/testify/assert"
)

type TestPlugin struct{}

func TestICanSetupAndTriggerPlugins(t *testing.T) {
	pl := plugins.NewPluginRecords()
	it := 0
	beforeTest := func(e plugins.Event, p any) {
		assert.Equal(t, plugins.OnUserCreation, e)
		assert.Equal(t, "quoi", p)
		it++
	}
	afterTest := func(e plugins.Event, p any) {
		assert.Equal(t, plugins.OnUserCreation, e)
		assert.Equal(t, "coubeh", p)
	}
	pl.Add("test-plugins", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins1", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins2", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins3", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins4", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins5", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins6", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins7", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins8", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins9", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins10", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins11", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins12", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins13", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins14", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	pl.Add("test-plugins15", tuple.Tuple3(plugins.OnUserCreation, beforeTest, afterTest))
	pl.TriggerBefore(plugins.OnUserCreation, "quoi")
	pl.TriggerAfter(plugins.OnUserCreation, "coubeh")
	assert.Equal(t, 136, it)
}
