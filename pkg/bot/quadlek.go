package bot

import (
	"context"

	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/data_store"
	"github.com/jirwin/quadlek/pkg/plugin_manager"
	"github.com/jirwin/quadlek/pkg/slack_manager"
	"github.com/jirwin/quadlek/pkg/webhook_manager"
)

type Config struct{}

func NewConfig() (Config, error) {
	return Config{}, nil
}

type QuadlekBot struct {
	l              *zap.Logger
	slackManager   slack_manager.Manager
	pluginManager  plugin_manager.Manager
	webhookManager webhook_manager.Manager
	c              Config
	dataStore      data_store.DataStore

	ctx    context.Context
	cancel context.CancelFunc
}

func (q *QuadlekBot) Start(ctx context.Context) error {
	go q.webhookManager.Run(ctx)
	go q.pluginManager.Run(ctx)

	q.ctx, q.cancel = context.WithCancel(ctx)

	err := q.slackManager.Start(ctx)
	if err != nil {
		q.l.Error("error initializing slack", zap.Error(err))
		return err
	}

	return nil
}

func (q *QuadlekBot) Stop() {
	q.dataStore.Close()
	q.cancel()
}

func (q *QuadlekBot) RegisterPlugin(plugin interface{}) error {
	return q.pluginManager.Register(plugin)
}

func New(
	c Config,
	l *zap.Logger,
	slackManager slack_manager.Manager,
	pluginManager plugin_manager.Manager,
	webhookManager webhook_manager.Manager,
	dataStore data_store.DataStore,
) (*QuadlekBot, error) {
	q := &QuadlekBot{
		c:              c,
		l:              l.Named("quadlek-bot"),
		slackManager:   slackManager,
		pluginManager:  pluginManager,
		webhookManager: webhookManager,
		dataStore:      dataStore,
	}

	return q, nil
}
