// This file should not be altered.
// @TODO: automatic regen of the file

package plugins

import (
	"log"

	"github.com/monkeydioude/goauth/pkg/data_types/tuple"
	"github.com/monkeydioude/goauth/pkg/plugins"
)

func AddPlugin(
	pluginName string,
	event plugins.Event,
	beforeHandler plugins.BeforeEventHandler,
	afterHandler plugins.AfterEventHandler,
) error {
	log.Printf("Added plugin '%s'", pluginName)
	return plugins.Plugins.Add(pluginName, tuple.Tuple3(event, beforeHandler, afterHandler))
}
