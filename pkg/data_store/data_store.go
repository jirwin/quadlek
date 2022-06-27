package data_store

import "github.com/jirwin/quadlek/pkg/data_store/boltdb"

type DataStore interface {
	InitPluginBucket(pluginID string) error
	// TODO(jirwin): This is an interface layering violation until plugins are properly refactored
	GetStore(pluginID string) boltdb.PluginStore
	Close()
}
