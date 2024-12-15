package plugins

import "errors"

var ErrPluginAlreadyExist = errors.New("plugin already exist")
var ErrPluginDoesNotExist = errors.New("plugin does not exist")
var ErrPluginInit = errors.New("plugin did not initialize")
var ErrLockAquire = errors.New("could not lock the mutex")
var ErrPluginPanicked = errors.New("plugin panicked")
var ErrPluginExecCancelTimeout = errors.New("plugin execution timeout")
var ErrEventUnknownStep = errors.New("unknown hook step")
var ErrHandlerWasNil = errors.New("handler was nil")
