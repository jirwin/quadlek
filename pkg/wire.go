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
	"github.com/jirwin/quadlek/pkg/slack_manager"
	"github.com/jirwin/quadlek/pkg/slack_manager/client"
	"github.com/jirwin/quadlek/pkg/uzap"
	"github.com/jirwin/quadlek/pkg/webhook_manager"
)

func NewQuadlek(ctx context.Context) (*bot.QuadlekBot, error) {
	wire.Build(
		uzap.Wired,

		boltdb.Wired,
		wire.Bind(new(data_store.DataStore), new(*boltdb.BoltDbStore)),

		client.Wired,
		wire.Bind(new(client.SlackClient), new(*client.SlackClientImpl)),

		slack_manager.Wired,
		wire.Bind(new(slack_manager.Manager), new(*slack_manager.ManagerImpl)),

		webhook_manager.Wired,
		wire.Bind(new(webhook_manager.Manager), new(*webhook_manager.ManagerImpl)),

		plugin_manager.Wired,
		wire.Bind(new(plugin_manager.Manager), new(*plugin_manager.ManagerImpl)),

		bot.Wired,
	)
	return nil, nil
}
