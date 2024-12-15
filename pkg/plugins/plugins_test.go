package plugins

import (
	"GOAuTh/pkg/data_types/tuple"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestICanHaveErrors(t *testing.T) {
	fakePlugins := PluginsRecord{
		record:          make([]plugin, 0),
		context:         context.TODO(),
		timeoutDuration: 1 * time.Second,
	}
	beforeTest := func(e Event, p any) {
		panic("oh no")
	}
	fakePlugins.Add("test-plugins", tuple.Tuple3[Event, BeforeEventHandler, AfterEventHandler](OnUserCreation, beforeTest, nil))
	if !errors.Is(fakePlugins.triggerEvent(Before, OnUserCreation, "quoi"), ErrPluginPanicked) {
		t.Error("should have panicked with error ErrPluginPanicked")
	}
	if !errors.Is(fakePlugins.triggerEvent("fake-step", OnUserCreation, ""), ErrEventUnknownStep) {
		t.Error("should have return ErrEventUnknownStep error")
	}
}

func TestICanHaveNilHandlers(t *testing.T) {
	fakePlugins := PluginsRecord{
		record:          make([]plugin, 0),
		context:         context.TODO(),
		timeoutDuration: 1 * time.Second,
	}
	fakePlugins.Add("test-plugins", tuple.Tuple3[Event, BeforeEventHandler, AfterEventHandler](OnUserCreation, nil, nil))
	// BeforeHandler is nil
	assert.NoError(t, fakePlugins.triggerEvent(Before, OnUserCreation, "quoi"))
	// AfterHandler is nil
	assert.NoError(t, fakePlugins.triggerEvent(After, OnUserCreation, "quoi"))
	if !errors.Is(fakePlugins.triggerEvent("fake-step", OnUserCreation, ""), ErrEventUnknownStep) {
		t.Error("should have return ErrEventUnknownStep error")
	}
}
