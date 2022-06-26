package plugin_manager

import "github.com/slack-go/slack/slackevents"

// dispatchReactions sends a reaction to all registered reaction hooks
func (m *ManagerImpl) dispatchReactions(ev *slackevents.ReactionAddedEvent) {
	for _, rh := range m.reactionHooks {
		rh.ReactionHook.Channel() <- &ReactionHookMsg{
			Bot:      b,
			Reaction: ev,
			Store:    b.getStore(rh.PluginId),
		}
	}
}
