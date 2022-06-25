package bot

import (
	"github.com/jirwin/quadlek/pkg/data_store"
	"github.com/jirwin/quadlek/pkg/plugin_manager"
	"github.com/jirwin/quadlek/pkg/slack"
	"github.com/jirwin/quadlek/pkg/webhook_server"
	"go.uber.org/zap"
)

type Config struct{}

type QuadlekBot struct {
	L             *zap.Logger
	Slack         slack.SlackAPI
	PluginManager plugin_manager.PluginManager
	WebhookServer webhook_server.WebhookServer
	C             Config
	DataStore     data_store.DataStore
}

func NewConfig() (Config, error) {
	return Config{}, nil
}
