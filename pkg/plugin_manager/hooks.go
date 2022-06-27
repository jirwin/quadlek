package plugin_manager

import (
	"github.com/slack-go/slack"
)

// dispatchHooks sends a slack message to all registered hooks
func (m *ManagerImpl) dispatchHooks(msg *slack.Msg) {
	for _, h := range m.hooks {
		h.Hook.Channel() <- &HookMsg{
			Helper: NewPluginHelper(h.PluginID, m.l, m.slackManager, m.dataStore.GetStore(h.PluginID)),
			Msg:    *msg,
		}
	}
}
