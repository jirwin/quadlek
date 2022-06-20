package quadlek

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

// PluginWebhook stores an incoming web request to be passed to a plugin
type PluginWebhook struct {
	Name           string
	Request        *http.Request
	ResponseWriter http.ResponseWriter
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
	_ = json.NewEncoder(w).Encode(obj)
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
	err := b.ValidateSlackRequest(r)
	if err != nil {
		b.Log.Error("failed validating request signature")
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}
	err = r.ParseForm()
	if err != nil {
		b.Log.Error("error parsing form. Invalid slack command hook.", zap.Error(err))
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	cmd := &slashCommand{}
	decoder.IgnoreUnknownKeys(true)
	err = decoder.Decode(cmd, r.PostForm)
	if err != nil {
		b.Log.Error("error marshalling slack command.", zap.Error(err))
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
					_, _ = w.Write([]byte{})
					return
				}

				prepareSlashCommandResp(resp)
				jsonResponse(w, resp)
			} else {
				<-timer.C
				_ = b.RespondToSlashCommand(cmd.ResponseUrl, resp)
			}
			return

		case <-timer.C:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte{})
			return
		}
	}
}

// handlePluginWebhook is an http handler that dispatches custom webhooks to the appropriate plugin
func (b *Bot) handlePluginWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	wh := b.GetWebhook(vars["webhook-name"])
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
		b.Log.Info("Webhook completed.")
	case <-time.After(time.Second * 5):
		b.Log.Info("Webhook timed out.")
	}
}

// WebhookServer starts a new http server that listens and responds to incoming webhooks.
// The Slack API uses webhooks for processing slash commands, and this server is used to respond to them.
// Plugins can also register custom webhooks that can be used however they choose. An example of this would be
// to process oauth2 callbacks to facilitate oauth2 flows for associating a user's slack account with an external service.
func (b *Bot) WebhookServer() {
	r := mux.NewRouter()
	r.HandleFunc("/slack/command", b.handleSlackCommand).Methods("POST")
	r.HandleFunc("/slack/plugin/{webhook-name}", b.handlePluginWebhook).Methods("GET", "POST", "DELETE", "PUT")
	r.HandleFunc("/slack/event", b.handleSlackEvent).Methods("POST")

	// TODO(jirwin): This listen address should be configurable
	srv := &http.Server{Addr: ":8000", Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			b.Log.Error("listen err", zap.Error(err))
		}
	}()

	<-b.ctx.Done()

	b.Log.Info("Shutting down webhook server")
	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	b.Log.Info("Shut down webhook server")
}

var InvalidRequestSignature = errors.New("invalid request signature")

// Validates the signature header for slack webhooks
func (b *Bot) ValidateSlackRequest(r *http.Request) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.Log.Error("error reading request body")
		return InvalidRequestSignature
	}

	rBody := ioutil.NopCloser(bytes.NewBuffer(body))
	r.Body = rBody

	ts := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")

	msg := fmt.Sprintf("v0:%s:%s", ts, body)
	key := []byte(b.verificationToken)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	checkSign := "v0=" + hex.EncodeToString(h.Sum(nil))

	if signature == checkSign {
		return nil
	}

	return errors.New("invalid request signature")
}
