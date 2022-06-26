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
		Bot:            b,
		Request:        r,
		ResponseWriter: w,
		Store:          b.getStore(wh.PluginId),
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
