package plugin_manager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/data_store"
)

type Config struct {
}

func NewConfig() (Config, error) {
	c := Config{}

	return c, nil
}

type Manager interface {
	Start(ctx context.Context)
	Close()
	Register(p interface{}) error
}

type ManagerImpl struct {
	c             Config
	l             *zap.Logger
	dataStore     data_store.DataStore
	commands      map[string]*registeredCommand
	webhooks      map[string]*registeredWebhook
	interactions  map[string]*registeredInteraction
	hooks         []*registeredHook
	reactionHooks []*registeredReactionHook
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func (m *ManagerImpl) Start(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (m *ManagerImpl) Close() {
	//TODO implement me
	panic("implement me")
}

// RegisterPlugin registers the given Plugin with the Bot.
func (m *ManagerImpl) Register(p interface{}) error {
	if p == nil {
		return fmt.Errorf("invalid plugin")
	}

	plugin, ok := p.(Plugin)
	if !ok {
		return errors.New("invalid plugin")
	}

	if plugin.GetId() == "" {
		return errors.New("Must provide a unique plugin id.")
	}

	err := m.dataStore.InitPluginBucket(plugin.GetId())
	if err != nil {
		return err
	}

	if lp, ok := plugin.(LoadPlugin); ok {
		err = lp.Load(b, m.dataStore.GetStore(lp.GetId()))
		if err != nil {
			return err
		}
	}

	if cp, ok := plugin.(CommandPlugin); ok {
		for _, command := range cp.GetCommands() {
			_, ok := m.commands[command.GetName()]
			if ok {
				return fmt.Errorf("Command already exists: %s", command.GetName())
			}
			m.commands[command.GetName()] = &registeredCommand{
				PluginId: cp.GetId(),
				Command:  command,
			}
			m.wg.Add(1)
			go func(c Command) {
				defer m.wg.Done()

				c.Run(m.ctx)
			}(command)
		}
	}

	if hp, ok := plugin.(HookPlugin); ok {
		for _, hook := range hp.GetHooks() {
			m.hooks = append(m.hooks, &registeredHook{
				PluginId: hp.GetId(),
				Hook:     hook,
			})
			m.wg.Add(1)
			go func(h Hook) {
				defer m.wg.Done()

				h.Run(m.ctx)
			}(hook)
		}
	}

	if rp, ok := plugin.(ReactionHookPlugin); ok {
		for _, reactionHook := range rp.GetReactionHooks() {
			m.reactionHooks = append(m.reactionHooks, &registeredReactionHook{
				PluginId:     rp.GetId(),
				ReactionHook: reactionHook,
			})
			m.wg.Add(1)
			go func(r ReactionHook) {
				defer m.wg.Done()

				r.Run(m.ctx)
			}(reactionHook)
		}
	}

	if wp, ok := plugin.(WebhookPlugin); ok {
		for _, wHook := range wp.GetWebhooks() {
			_, ok := m.webhooks[wHook.GetName()]
			if ok {
				return fmt.Errorf("Webhook already exists: %s", wHook.GetName())
			}
			m.webhooks[wHook.GetName()] = &registeredWebhook{
				PluginId: wp.GetId(),
				Webhook:  wHook,
			}
			m.wg.Add(1)
			go func(wh Webhook) {
				defer b.wg.Done()

				wh.Run(b.ctx)
			}(wHook)
		}
	}

	if ip, ok := plugin.(InteractionPlugin); ok {
		for _, ic := range ip.GetInteractions() {
			_, ok := m.interactions[ic.GetName()]
			if ok {
				return fmt.Errorf("Interaction plugin already exists:  %s", ic.GetName())
			}
			m.interactions[ic.GetName()] = &registeredInteraction{
				PluginId:    ip.GetId(),
				Interaction: ic,
			}
			m.wg.Add(1)
			go func(s Interaction) {
				defer b.wg.Done()
				s.Run(m.ctx)
			}(ic)
		}
	}

	return nil
}

func New(c Config, l *zap.Logger, dataStore data_store.DataStore) (*ManagerImpl, error) {
	m := &ManagerImpl{
		c:         c,
		l:         l,
		dataStore: dataStore,
	}

	return m, nil
}
