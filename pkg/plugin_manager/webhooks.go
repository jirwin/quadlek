package plugin_manager

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// PluginWebhook stores an incoming web request to be passed to a plugin
type PluginWebhook struct {
	Name           string
	Request        *http.Request
	ResponseWriter http.ResponseWriter
}

// dispatchWebhook parses an incoming webhook and sends it to the plugin it is registered to
func (m *ManagerImpl) dispatchWebhook(webhook *PluginWebhook) {
	wh := m.getWebhook(webhook.Name)
	if wh == nil {
		return
	}

	wh.Webhook.Channel() <- &WebhookMsg{
		Helper:         NewPluginHelper(wh.PluginID, m.l, m.slackManager, m.dataStore.GetStore(wh.PluginID)),
		Request:        webhook.Request,
		ResponseWriter: webhook.ResponseWriter,
	}
}

func (m *ManagerImpl) getWebhook(webhookName string) *registeredWebhook {
	if webhookName == "" {
		return nil
	}

	if wh, ok := m.webhooks[webhookName]; ok {
		return wh
	}

	return nil
}

// handlePluginWebhook is an http handler that dispatches custom webhooks to the appropriate plugin
func (m *ManagerImpl) handlePluginWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	wh := m.getWebhook(vars["webhook-name"])
	if wh == nil {
		return
	}

	done := make(chan bool, 1)
	msg := &WebhookMsg{
		Helper:         NewPluginHelper(wh.PluginID, m.l, m.slackManager, m.dataStore.GetStore(wh.PluginID)),
		Request:        r,
		ResponseWriter: w,
		Done:           done,
	}
	wh.Webhook.Channel() <- msg

	select {
	case <-done:
		m.l.Info("Webhook completed.")
	case <-time.After(time.Second * 5):
		m.l.Info("Webhook timed out.")
	}
}
