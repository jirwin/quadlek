package plugin_manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/data_store/boltdb"
	"github.com/jirwin/quadlek/pkg/slack_manager"
)

type PluginHelper interface {
	StopBot()
	OpenView(triggerID string, response slack.ModalViewRequest) (*slack.ViewResponse, error)
	GetMessageLog(channel string, opts MessageLotOpts) ([]slack.Message, error)
	GetMessage(channel, ts string) (slack.Message, error)
	RespondToSlashCommand(url string, cmdResp *CommandResp) error
	Respond(msg slack.Msg, resp string)
	Say(channel string, resp string)
	GetUser(userId string) (slack.User, error)
	GetChannel(chanId string) (slack.Channel, error)
	GetChannelId(channelName string) (string, error)
	Store() boltdb.PluginStore
}

type pluginHelper struct {
	slackManager slack_manager.Manager
	store        boltdb.PluginStore
	l            *zap.Logger
	pluginID     string
}

func (p *pluginHelper) StopBot() {
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
}

func (p *pluginHelper) Store() boltdb.PluginStore {
	return p.store
}

func (p *pluginHelper) OpenView(triggerID string, response slack.ModalViewRequest) (*slack.ViewResponse, error) {
	r, err := p.slackManager.Slack().Api().OpenView(triggerID, response)
	if err != nil {
		p.l.Error("error opening view", zap.Error(err))
		return nil, err
	}

	return r, nil
}

// MessageLotOpts is the struct that you use to configure what messages you want to retrieve from the API.
// IncludeBots: If true, include messages from bots(not just pkg bots)
// Count: The max number of messages to return
// Period: The amount of time to look backwards when looking for messages
// SkipAttachments: If true, don't return message attachments.
type MessageLotOpts struct {
	IncludeBots     bool
	Count           int
	Period          time.Duration
	SkipAttachments bool
}

func (p *pluginHelper) GetMessageLog(channel string, opts MessageLotOpts) ([]slack.Message, error) {
	params := &slack.GetConversationHistoryParameters{}
	if opts.Count != 0 {
		params.Limit = opts.Count
	}
	if opts.Period != time.Duration(0) {
		oldest := time.Now().UTC().Add(opts.Period*-1).UnixNano() / 1000000
		oldestTs := fmt.Sprintf("%d.000", oldest)
		params.Oldest = oldestTs
	}
	params.ChannelID = channel
	history, err := p.slackManager.Slack().Api().GetConversationHistory(params)
	if err != nil {
		return nil, err
	}

	msgs := []slack.Message{}
	for _, msg := range history.Messages {
		if !opts.IncludeBots && msg.SubType == "bot_message" {
			continue
		}

		if opts.SkipAttachments && len(msg.Attachments) != 0 {
			continue
		}

		if msg.SubType != "" {
			continue
		}

		msgs = append(msgs, msg)
	}

	return msgs, nil
}

func (p *pluginHelper) GetMessage(channel, ts string) (slack.Message, error) {
	params := &slack.GetConversationHistoryParameters{}
	params.Limit = 1
	params.Latest = ts
	params.Inclusive = true
	params.ChannelID = channel

	history, err := p.slackManager.Slack().Api().GetConversationHistory(params)
	if err != nil {
		return slack.Message{}, err
	}

	if len(history.Messages) != 1 {
		return slack.Message{}, err
	}

	return history.Messages[0], nil
}

func (p *pluginHelper) RespondToSlashCommand(url string, cmdResp *CommandResp) error {
	prepareSlashCommandResp(cmdResp)

	jsonBytes, err := json.Marshal(cmdResp)
	if err != nil {
		p.l.Error("error marshalling json.", zap.Error(err))
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
		p.l.Error("error responding to slash command.", zap.Error(err))
		return err
	}
	return nil
}

func (p *pluginHelper) Respond(msg slack.Msg, resp string) {
	p.slackManager.Slack().Api().PostMessage(
		msg.Channel,
		slack.MsgOptionText(fmt.Sprintf("<@%s>: %s", msg.User, resp), false),
	) //nolint:errcheck
}

func (p *pluginHelper) Say(channel string, resp string) {
	p.slackManager.Slack().Api().PostMessage(channel, slack.MsgOptionText(resp, false)) //nolint:errcheck

}

func (p *pluginHelper) GetUser(userID string) (slack.User, error) {
	return p.slackManager.GetUser(userID)
}

func (p *pluginHelper) GetChannel(chanID string) (slack.Channel, error) {
	return p.slackManager.GetChannel(chanID)
}

func (p *pluginHelper) GetChannelId(channelName string) (string, error) {
	return p.slackManager.GetChannelId(channelName)
}

func NewPluginHelper(pluginID string, l *zap.Logger, slackManager slack_manager.Manager, store boltdb.PluginStore) *pluginHelper {
	ph := &pluginHelper{
		slackManager: slackManager,
		l:            l.Named(fmt.Sprintf("plugin-helper-%s", pluginID)),
		pluginID:     pluginID,
		store:        store,
	}

	return ph
}
