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

func (sc *slashCommand) Reply() chan<- *CommandResp {
	return sc.responseChan
}

type slashCommandErrorResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
}

func jsonResponse(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(obj)
}

func generateErrorMsg(w http.ResponseWriter, msg string) {
	resp := &slashCommandErrorResponse{
		ResponseType: "ephemeral",
		Text:         msg,
	}

	jsonResponse(w, resp)
}

func (b *Bot) WebhookServer() {
	r := mux.NewRouter()
	r.HandleFunc("/slack/command", b.handleSlackCommand).Methods("POST")

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
