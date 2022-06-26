package plugin_manager

import "go.uber.org/zap"

type Config struct {
}

func NewConfig() (Config, error) {
	c := Config{}

	return c, nil
}

type Manager interface {
}

type ManagerImpl struct {
	C Config
	L *zap.Logger
}
