package data_store

type DataStore interface {
	InitPluginBucket(pluginID string) error
	GetStore(pluginID string)
	Close()
}
