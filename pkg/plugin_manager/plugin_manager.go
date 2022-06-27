package plugin_manager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/data_store"
	"github.com/jirwin/quadlek/pkg/slack_manager"
	"github.com/jirwin/quadlek/pkg/webhook_manager"
)

type Config struct {
}

func NewConfig() (Config, error) {
	c := Config{}

	return c, nil
}

type Manager interface {
	Run(ctx context.Context)
	Register(p interface{}) error
	RespondToSlashCommand(url string, cmdResp *CommandResp) error
}

type ManagerImpl struct {
	c                    Config
	l                    *zap.Logger
	webhookManager       webhook_manager.Manager
	slackManager         slack_manager.Manager
	dataStore            data_store.DataStore
	commands             map[string]*registeredCommand
	webhooks             map[string]*registeredWebhook
	interactions         map[string]*registeredInteraction
	cmdChannel           chan *slashCommand
	pluginWebhookChannel chan *PluginWebhook
	interactionChannel   chan *slack.InteractionCallback
	hooks                []*registeredHook
	reactionHooks        []*registeredReactionHook
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   sync.WaitGroup
	running              bool
}

func (m *ManagerImpl) Run(ctx context.Context) {
	m.running = true

	m.ctx, m.cancel = context.WithCancel(ctx)
	defer m.cancel()

	go m.handleEvents(m.ctx)

	m.webhookManager.RegisterRoute("/slack/command", m.handleSlackCommand, []string{"POST"}, true)
	m.webhookManager.RegisterRoute("/slack/plugin/{webhook-name}", m.handlePluginWebhook, []string{"GET", "POST", "DELETE", "PUT"}, false)
	m.webhookManager.RegisterRoute("/slack/interaction", m.handleSlackInteraction, []string{"POST"}, true)
	m.webhookManager.RegisterRoute("/slack/event", m.handleSlackEvent, []string{"POST"}, true)
	m.l.Info("running plugin manager")

	<-m.ctx.Done()
	m.running = false
}

func (m *ManagerImpl) handleEvents(ctx context.Context) {
	for {
		select {
		// Slash Command
		case slashCmd := <-m.cmdChannel:
			m.l.Info("dispatching slash command", zap.String("command", slashCmd.Command))
			m.dispatchCommand(slashCmd)

		// Custom webhook
		case wh := <-m.pluginWebhookChannel:
			m.dispatchWebhook(wh)

		// Interaction
		case ic := <-m.interactionChannel:
			m.dispatchInteraction(ic)

		case <-ctx.Done():
			return
		}
	}
}

// Register registers the given Plugin with the Bot.
func (m *ManagerImpl) Register(p interface{}) error {
	if !m.running {
		return fmt.Errorf("bot must be running to register plugins")
	}
	if p == nil {
		return fmt.Errorf("invalid plugin")
	}

	plgin, ok := p.(Plugin)
	if !ok {
		return errors.New("invalid plugin")
	}

	if plgin.GetId() == "" {
		return errors.New("Must provide a unique plugin id.")
	}

	err := m.dataStore.InitPluginBucket(plgin.GetId())
	if err != nil {
		return err
	}

	if lp, ok := plgin.(LoadPlugin); ok {
		err = lp.Load(NewPluginHelper(plgin.GetId(), m.l, m.slackManager, m.dataStore.GetStore(plgin.GetId())))
		if err != nil {
			return err
		}
	}

	if cp, ok := plgin.(CommandPlugin); ok {
		for _, cmd := range cp.GetCommands() {
			_, ok := m.commands[cmd.GetName()]
			if ok {
				return fmt.Errorf("Command already exists: %s", cmd.GetName())
			}

			m.l.Info("registering command", zap.String("command_name", cmd.GetName()), zap.String("plugin_id", cp.GetId()))

			m.commands[cmd.GetName()] = &registeredCommand{
				PluginID: cp.GetId(),
				Command:  cmd,
			}
			m.wg.Add(1)
			go func(c Command) {
				defer m.wg.Done()
				c.Run(m.ctx)
			}(cmd)
		}
	}

	if hp, ok := plgin.(HookPlugin); ok {
		for _, hk := range hp.GetHooks() {
			m.hooks = append(m.hooks, &registeredHook{
				PluginID: hp.GetId(),
				Hook:     hk,
			})

			m.l.Info("registering hook", zap.String("plugin_id", hp.GetId()))

			m.wg.Add(1)
			go func(h Hook) {
				defer m.wg.Done()

				h.Run(m.ctx)
			}(hk)
		}
	}

	if rp, ok := plgin.(ReactionHookPlugin); ok {
		for _, rh := range rp.GetReactionHooks() {
			m.reactionHooks = append(m.reactionHooks, &registeredReactionHook{
				PluginID:     rp.GetId(),
				ReactionHook: rh,
			})

			m.l.Info("registering reaction hook", zap.String("plugin_id", rp.GetId()))

			m.wg.Add(1)
			go func(r ReactionHook) {
				defer m.wg.Done()

				r.Run(m.ctx)
			}(rh)
		}
	}

	if wp, ok := plgin.(WebhookPlugin); ok {
		for _, wHook := range wp.GetWebhooks() {
			_, ok := m.webhooks[wHook.GetName()]
			if ok {
				return fmt.Errorf("Webhook already exists: %s", wHook.GetName())
			}
			m.webhooks[wHook.GetName()] = &registeredWebhook{
				PluginID: wp.GetId(),
				Webhook:  wHook,
			}
			m.l.Info("registering webhook", zap.String("webhook_name", wHook.GetName()), zap.String("plugin_id", wp.GetId()))

			m.wg.Add(1)
			go func(wh Webhook) {
				defer m.wg.Done()

				wh.Run(m.ctx)
			}(wHook)
		}
	}

	if ip, ok := plgin.(InteractionPlugin); ok {
		for _, ic := range ip.GetInteractions() {
			_, ok := m.interactions[ic.GetName()]
			if ok {
				return fmt.Errorf("Interaction plugin already exists:  %s", ic.GetName())
			}
			m.interactions[ic.GetName()] = &registeredInteraction{
				PluginID:    ip.GetId(),
				Interaction: ic,
			}

			m.l.Info("registering hook", zap.String("interaction_name", ic.GetName()), zap.String("plugin_id", ip.GetId()))

			m.wg.Add(1)
			go func(s Interaction) {
				defer m.wg.Done()
				s.Run(m.ctx)
			}(ic)
		}
	}

	return nil
}

func New(
	c Config,
	l *zap.Logger,
	webhookManager webhook_manager.Manager,
	slackManager slack_manager.Manager,
	dataStore data_store.DataStore,
) (*ManagerImpl, error) {
	m := &ManagerImpl{
		c:                    c,
		l:                    l.Named("plugin-manager"),
		webhookManager:       webhookManager,
		slackManager:         slackManager,
		dataStore:            dataStore,
		cmdChannel:           make(chan *slashCommand),
		pluginWebhookChannel: make(chan *PluginWebhook),
		interactionChannel:   make(chan *slack.InteractionCallback),
		commands:             make(map[string]*registeredCommand),
		webhooks:             make(map[string]*registeredWebhook),
		interactions:         make(map[string]*registeredInteraction),
	}

	return m, nil
}
