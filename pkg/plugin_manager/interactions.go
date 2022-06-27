package plugin_manager

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// getInteraction returns the registeredInteraction for the given CallbackID.
// Matches on the first *- e.g. 'restart', 'restart-', 'restart-modal' will be sent to the 'restart' plugin.
// 'celebrate-read' will be sent to the 'celebrate' plugin.
func (m *ManagerImpl) getInteraction(callbackID string) *registeredInteraction {
	if callbackID == "" {
		return nil
	}

	callbackParts := strings.Split(callbackID, "-")

	if interact, ok := m.interactions[callbackParts[0]]; ok {
		return interact
	}

	return nil
}

// dispatchWebhook parses an incoming webhook and sends it to the plugin it is registered to
func (m *ManagerImpl) dispatchInteraction(cb *slack.InteractionCallback) {
	callbackID := ""
	switch cb.Type {
	case "view_submission":
		callbackID = cb.View.CallbackID
	default:
		callbackID = cb.CallbackID
	}

	ic := m.getInteraction(callbackID)
	if ic == nil {
		return
	}

	ic.Interaction.Channel() <- &InteractionMsg{
		Helper:      NewPluginHelper(ic.PluginID, m.l, m.slackManager, m.dataStore.GetStore(ic.PluginID)),
		Interaction: *cb,
	}
}

func (m *ManagerImpl) handleSlackInteraction(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.l.Error("error parsing interaction", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte{})
		return
	}

	ev := &slack.InteractionCallback{}
	err = json.Unmarshal([]byte(r.Form.Get("payload")), &ev)
	if err != nil {
		m.l.Error("invalid interaction json", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte{})
		return
	}

	if ev.Type == "" {
		m.l.Error("missing interaction type")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte{})
		return
	}

	m.interactionChannel <- ev

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte{})
	return
}
