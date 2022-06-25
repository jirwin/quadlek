//go:build wireinject
// +build wireinject

package quadlek

import (
	"context"
	"github.com/google/wire"
	"github.com/jirwin/quadlek/pkg/bot"
	"github.com/jirwin/quadlek/pkg/data_store"
	"github.com/jirwin/quadlek/pkg/data_store/boltdb"
	"github.com/jirwin/quadlek/pkg/plugin_manager"
	"github.com/jirwin/quadlek/pkg/slack"
	"github.com/jirwin/quadlek/pkg/uzap"
	"github.com/jirwin/quadlek/pkg/webhook_server"
)

func NewQuadlek(ctx context.Context) (*bot.QuadlekBot, error) {
	wire.Build(
		uzap.Wired,

		slack.Wired,
		wire.Bind(new(slack.SlackAPI), new(*slack.SlackClient)),

		boltdb.Wired,
		wire.Bind(new(data_store.DataStore), new(*boltdb.BoltDbStore)),

		plugin_manager.Wired,
		wire.Bind(new(plugin_manager.PluginManager), new(*plugin_manager.Manager)),

		webhook_server.Wired,
		wire.Bind(new(webhook_server.WebhookServer), new(*webhook_server.Server)),

		bot.Wired,
	)
	return nil, nil
}
