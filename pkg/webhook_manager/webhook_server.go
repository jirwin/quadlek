package webhook_manager

import (
	"fmt"
	"github.com/jirwin/quadlek/pkg/plugin_manager"
	"go.uber.org/zap"
	"net/http"
	"os"
)

type Config struct {
	ListenAddress string
}

type Manager interface{}

type ManagerImpl struct {
	L             *zap.Logger
	C             Config
	PluginManager plugin_manager.Manager
	server        *http.Server
}

func NewConfig() (Config, error) {
	c := Config{}

	listenAddr := os.Getenv("QUADLEK_WEBHOOK_LISTEN_ADDR")
	if listenAddr == "" {
		return Config{}, fmt.Errorf("QUADLEK_WEBHOOK_LISTEN_ADDR must be set e.g. 0.0.0.0:8000")
	}
	c.ListenAddress = listenAddr

	return c, nil
}

func New(c Config, l *zap.Logger, pluginManager plugin_manager.Manager) (*ManagerImpl, error) {
	ws := &ManagerImpl{
		L:             l,
		C:             c,
		PluginManager: pluginManager,
	}

	ws.server = &http.Server{Addr: c.ListenAddress}

	return ws, nil
}
