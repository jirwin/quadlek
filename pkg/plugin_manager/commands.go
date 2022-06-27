package plugin_manager

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

var decoder = schema.NewDecoder()

// slashCommand is an internal object that parses slash command webhooks coming from the slackManager servers
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

// dispatchCommand parses an incoming slash command and sends it to the plugin it is registered to
func (m *ManagerImpl) dispatchCommand(slashCmd *slashCommand) {
	if slashCmd.Command == "" {
		return
	}
	cmdName := slashCmd.Command[1:]

	m.l.Info("dispatched command", zap.String("cmd_name", cmdName))

	cmd := m.getCommand(cmdName)
	if cmd == nil {
		return
	}

	m.l.Info("fetched command for dispatch", zap.String("plugin_id", cmd.PluginID))

	cmd.Command.Channel() <- &CommandMsg{
		Helper:  NewPluginHelper(cmd.PluginID, m.l, m.slackManager, m.dataStore.GetStore(cmd.PluginID)),
		Command: slashCmd,
	}
}

// GetCommand returns the registeredCommand for the provided command name
func (m *ManagerImpl) getCommand(cmdText string) *registeredCommand {
	if cmdText == "" {
		return nil
	}

	if cmd, ok := m.commands[cmdText]; ok {
		return cmd
	}

	return nil
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

// prepareSlashCommandResp prepares a command response for API submission
func prepareSlashCommandResp(cmd *CommandResp) {
	if cmd.ResponseType == "" {
		if cmd.InChannel {
			cmd.ResponseType = "in_channel"
		} else {
			cmd.ResponseType = "ephemeral"
		}
	}
}

// handleSlackCommand is an http handler that parses an incoming slash command webhook
// and dispatches it to the proper plugin.
// It attempts to handle responding to the request if the plugin doesn't respond in time.
func (m *ManagerImpl) handleSlackCommand(w http.ResponseWriter, r *http.Request) {
	m.l.Info("handling slack command")
	err := r.ParseForm()
	if err != nil {
		m.l.Error("error parsing form. Invalid slack command hook.", zap.Error(err))
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	cmd := &slashCommand{}
	decoder.IgnoreUnknownKeys(true)
	err = decoder.Decode(cmd, r.PostForm)
	if err != nil {
		m.l.Error("error marshalling slack command.", zap.Error(err))
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	m.l.Info("sending command to channel")
	respChan := make(chan *CommandResp)
	cmd.responseChan = respChan
	m.cmdChannel <- cmd
	m.l.Info("sent command")
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
				_ = m.RespondToSlashCommand(cmd.ResponseUrl, resp)
			}
			return

		case <-timer.C:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte{})
			return
		}
	}
}

// RespondToSlashCommand sends a command response to the slack API in order to respond to a slash command.
func (m *ManagerImpl) RespondToSlashCommand(url string, cmdResp *CommandResp) error {
	prepareSlashCommandResp(cmdResp)

	jsonBytes, err := json.Marshal(cmdResp)
	if err != nil {
		m.l.Error("error marshalling json", zap.Error(err))
		return err
	}
	data := bytes.NewBuffer(jsonBytes)
	err = json.NewEncoder(data).Encode(cmdResp)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err != nil {
		m.l.Error("error responding to slash command", zap.Error(err))
		return err
	}
	return nil
}
