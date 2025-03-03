// This file should not be altered.
// @TODO: automatic regen of the file

package plugins

import (
	"GOAuTh/pkg/data_types/tuple"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

type BeforeEventHandler = func(event Event, payload any)
type AfterEventHandler = func(event Event, payload any)

type plugin struct {
	name     string
	handlers tuple.Tuple_3[Event, BeforeEventHandler, AfterEventHandler]
}

type PluginsRecord struct {
	record          []plugin
	mutex           sync.Mutex
	context         context.Context
	timeoutDuration time.Duration
}

func NewPluginRecords() PluginsRecord {
	return PluginsRecord{
		record:          make([]plugin, 0),
		context:         context.Background(),
		timeoutDuration: 5 * time.Second,
	}
}

var Plugins = NewPluginRecords()

func (pr *PluginsRecord) Add(
	pluginName string,
	handlers tuple.Tuple_3[Event, BeforeEventHandler, AfterEventHandler],
) error {
	if pr.Exist(pluginName) {
		return fmt.Errorf("PluginsRecord.Add: %w: %s", ErrPluginAlreadyExist, pluginName)
	}
	if !pr.mutex.TryLock() {
		return ErrLockAquire
	}
	defer pr.mutex.Unlock()
	pr.record = append(pr.record, plugin{
		name:     pluginName,
		handlers: handlers,
	})
	return nil
}

func (pr *PluginsRecord) Exist(pluginName string) bool {
	return slices.ContainsFunc(pr.record, func(trial plugin) bool { return trial.name == pluginName })
}

func prepareExec(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc, chan error) {
	errChan := make(chan error)
	ctx, cancelFn := context.WithTimeout(ctx, timeout)
	return ctx, cancelFn, errChan
}

func (pr *PluginsRecord) execPluginHook(
	plugin plugin,
	step Step,
	errChan chan error,
	event Event,
	payload any,
) {
	defer func() {
		if r := recover(); r != nil {
			errChan <- ErrPluginPanicked
		}
	}()
	pr.mutex.Lock()
	defer pr.mutex.Unlock()
	switch step {
	case "before":
		if plugin.handlers.B == nil {
			break
		}
		plugin.handlers.B(event, payload)
	case "after":
		if plugin.handlers.C == nil {
			break
		}
		plugin.handlers.C(event, payload)
	default:
		errChan <- ErrEventUnknownStep
	}
	errChan <- nil
}

func handleResult(
	plugin plugin,
	step Step,
	ctx context.Context,
	timeout time.Duration,
	errChan chan error,
) error {
	select {
	case errValue := <-errChan:
		if errValue != nil {
			return fmt.Errorf("%s event hook: %s: %w", step, plugin.name, errValue)
		}
	case <-ctx.Done():
		return fmt.Errorf("%s event hook: %s: %w after %fs", step, plugin.name, ErrPluginExecCancelTimeout, timeout.Seconds())
	}
	return nil
}

func (pr *PluginsRecord) triggerEvent(step Step, event Event, payload any) error {
	var err error

	for _, plugin := range pr.record {
		ctx, cancelFn, errChan := prepareExec(pr.context, pr.timeoutDuration)
		go pr.execPluginHook(plugin, step, errChan, event, payload)
		err = errors.Join(err, handleResult(plugin, step, ctx, pr.timeoutDuration, errChan))
		cancelFn()
	}
	return err
}

func (pr *PluginsRecord) TriggerBefore(event Event, payload any) error {
	return pr.triggerEvent(Before, event, payload)
}

func (pr *PluginsRecord) TriggerAfter(event Event, payload any) error {
	return pr.triggerEvent(After, event, payload)
}
