package plugin_manager

import (
	"github.com/slack-go/slack/slackevents"
)

// dispatchReactions sends a reaction to all registered reaction hooks
func (m *ManagerImpl) dispatchReactions(ev *slackevents.ReactionAddedEvent) {
	for _, rh := range m.reactionHooks {
		rh.ReactionHook.Channel() <- &ReactionHookMsg{
			Helper:   NewPluginHelper(rh.PluginID, m.l, m.slackManager, m.dataStore.GetStore(rh.PluginID)),
			Reaction: ev,
		}
	}
}
