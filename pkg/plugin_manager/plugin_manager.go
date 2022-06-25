package plugin_manager

import "go.uber.org/zap"

type Config struct {
}

func NewConfig() (Config, error) {
	c := Config{}

	return c, nil
}

type PluginManager interface {
}

type Manager struct {
	C Config
	L *zap.Logger
}
