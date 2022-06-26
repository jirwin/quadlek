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
	SlackManager   slack_manager.Manager
	PluginManager  plugin_manager.Manager
	WebhookManager webhook_manager.Manager
	c              Config
	DataStore      data_store.DataStore

	ctx    context.Context
	cancel context.CancelFunc
}

func (q *QuadlekBot) Start(ctx context.Context) error {
	go q.WebhookManager.Start()
	go q.PluginManager.Start()

	q.ctx, q.cancel = context.WithCancel(ctx)

	err := q.SlackManager.Init()
	if err != nil {
		q.l.Error("error initializing slack", zap.Error(err))
		return err
	}

	return nil
}

func (q *QuadlekBot) Stop() {
	q.cancel()
	q.wg.Wait()
	if q.DataStore != nil {
		q.DataStore.Close()
	}
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
		SlackManager:   slackManager,
		PluginManager:  pluginManager,
		WebhookManager: webhookManager,
		DataStore:      dataStore,
	}

	return q, nil
}
