// This file should not be altered.
// @TODO: automatic regen of the file

package plugins

import (
	"GOAuTh/pkg/data_types/tuple"
	"GOAuTh/pkg/plugins"
	"log"
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
