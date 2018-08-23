package quadlek

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

// PluginWebhook stores an incoming web request to be passed to a plugin
type PluginWebhook struct {
	Name    string
	Request *http.Request
}

// slashCommand is an internal object that parses slash command webhooks coming from the Slack servers
type slashCommand struct {
	Token        string            `schema:"token"`
	TeamId       string            `schema:"team_id"`
	TeamDomain   string            `schema:"team_domain"`
	ChannelId    string            `schema:"channel_id"`
	ChannelName  string            `schema:"channel_name"`
	UserId       string            `schema:"user_id"`
	UserName     string            `schema:"user_name"`
	Command      string            `schema:"command"`
	Text         string            `schema:"text"`
	ResponseUrl  string            `schema:"response_url"`
	responseChan chan *CommandResp `schema:"-"`
}

// Reply returns the channel to write command responses to.
func (sc *slashCommand) Reply() chan<- *CommandResp {
	return sc.responseChan
}

// slashCommandErrorResponse is used to return an error to the user when a slash command can't be completed successfully
type slashCommandErrorResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
}

// jsonResponse encodes a generic object to json and writes it to the provided HTTP response
func jsonResponse(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(obj)
}

// generateErrorMsg encodes a slashCommandErrorResponse to json and writes it to the provided HTTP response
func generateErrorMsg(w http.ResponseWriter, msg string) {
	resp := &slashCommandErrorResponse{
		ResponseType: "ephemeral",
		Text:         msg,
	}

	jsonResponse(w, resp)
}

// handleSlackCommand is an http handler that parses an incoming slash command webhook
// and dispatches it to the proper plugin.
// It attempts to handle responding to the request if the plugin doesn't respond in time.
func (b *Bot) handleSlackCommand(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error parsing form. Invalid slack command hook.")
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	cmd := &slashCommand{}
	decoder.IgnoreUnknownKeys(true)
	err = decoder.Decode(cmd, r.PostForm)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error marshalling slack command.")
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	if cmd.Token != b.verificationToken {
		log.Error("Invalid validation token was used. Ignoring.")
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	respChan := make(chan *CommandResp)
	cmd.responseChan = respChan
	b.cmdChannel <- cmd

	timer := time.NewTimer(time.Millisecond * 2500)
	for {
		select {
		case resp := <-respChan:
			if timer.Stop() {
				// Got a nil response, so the plugin is explicitly not sending a response here and will send one manually.
				if resp == nil {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte{})
					return
				}

				prepareSlashCommandResp(resp)
				jsonResponse(w, resp)
			} else {
				<-timer.C
				b.RespondToSlashCommand(cmd.ResponseUrl, resp)
			}
			return

		case <-timer.C:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{})
			return
		}
	}
}

// handlePluginWebhook is an http handler that dispatches custom webhooks to the appropriate plugin
func (b *Bot) handlePluginWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	b.pluginWebhookChannel <- &PluginWebhook{
		Name:    vars["webhook-name"],
		Request: r,
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

// WebhookServer starts a new http server that listens and responds to incoming webhooks.
// The Slack API uses webhooks for processing slash commands, and this server is used to respond to them.
// Plugins can also register custom webhooks that can be used however they choose. An example of this would be
// to process oauth2 callbacks to facilitate oauth2 flows for associating a user's slack account with an external service.
func (b *Bot) WebhookServer() {
	r := mux.NewRouter()
	r.HandleFunc("/slack/command", b.handleSlackCommand).Methods("POST")
	r.HandleFunc("/slack/plugin/{webhook-name}", b.handlePluginWebhook).Methods("GET")

	// TODO(jirwin): This listen address should be configurable
	srv := &http.Server{Addr: ":8000", Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	<-b.ctx.Done()

	log.Info("Shutting down webhook server")
	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	log.Info("Shut down webhook server")
}
