// quadlek is a slack Bot that is built on top of the nlopes Slack client.
//
// For a good source of examples, look at the included plugins at https://github.com/jirwin/quadlek/tree/master/plugins.
//
// Read more about the client and Slack APIs at: https://github.com/nlopes/slack and https://api.slack.com
package quadlek

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"context"

	"sync"

	"math/rand"

	"errors"

	"github.com/boltdb/bolt"
	"github.com/slack-go/slack"
)

// This is the core struct for the Bot, and provides all methods required for interacting with various Slack APIs.
//
// An instance of the bot is provided to plugins to enable plugins to interact with the Slack API.
type Bot struct {
	Log                  *zap.Logger
	apiKey               string
	verificationToken    string
	api                  *slack.Client
	channels             map[string]slack.Channel
	humanChannels        map[string]slack.Channel
	userId               string
	botId                string
	humanUsers           map[string]slack.User
	users                map[string]slack.User
	commands             map[string]*registeredCommand
	cmdChannel           chan *slashCommand
	webhooks             map[string]*registeredWebhook
	pluginWebhookChannel chan *PluginWebhook
	hooks                []*registeredHook
	reactionHooks        []*registeredReactionHook
	db                   *bolt.DB
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   sync.WaitGroup
}

// GetUserId returns the Slack user ID for the Bot.
func (b *Bot) GetUserId() string {
	return b.userId
}

// GetBotId returns the Slack bot ID
func (b *Bot) GetBotId() string {
	return b.botId
}

// GetApi returns the Slack API client.
// You can use this client to perform actions that use the Slack Web API.
// See https://api.slack.com/web for more details.
func (b *Bot) GetApi() *slack.Client {
	return b.api
}

// GetChannelId returns the Slack channel ID for a given human-readable channel name.
func (b *Bot) GetChannelId(chanName string) (string, error) {
	channel, ok := b.humanChannels[chanName]
	if !ok {
		return "", errors.New("Channel not found.")
	}

	return channel.ID, nil
}

// GetChannel returns the Slack channel object given a channel ID
func (b *Bot) GetChannel(chanId string) (*slack.Channel, error) {
	channel, ok := b.channels[chanId]
	if !ok {
		return nil, errors.New("Channel not found.")
	}

	return &channel, nil
}

// GetUser returns the Slack user object given a user ID
func (b *Bot) GetUser(userId string) (*slack.User, error) {
	user, ok := b.users[userId]
	if !ok {
		return nil, errors.New("User not found.")
	}

	return &user, nil
}

// GetUserName returns the human-readable user name for a given user ID
func (b *Bot) GetUserName(userId string) (string, error) {
	user, ok := b.users[userId]
	if !ok {
		return "", errors.New("User not found.")
	}

	return user.Name, nil
}

// Respond responds to a Slack message
// The sent message will go to the same channel as the message that is being responded to and will highlight
// the author of the original message.
func (b *Bot) Respond(msg *slack.Msg, resp string) {
	b.api.PostMessage(msg.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s>: %s", msg.User, resp), false)) //nolint:errcheck
}

// PostMessage sends a new message to Slack using the provided channel and message string.
// It returns the channel ID the message was posted to, and the timestamp that the message was posted at.
// In combination these can be used to identify the exact message that was sent.
func (b *Bot) PostMessage(channel, resp string, params slack.PostMessageParameters) (string, string, error) {
	return b.api.PostMessage(channel, slack.MsgOptionText(resp, false))
}

// Say sends a message to the provided channel
func (b *Bot) Say(channel string, resp string) {
	b.api.PostMessage(channel, slack.MsgOptionText(resp, false)) //nolint:errcheck
}

// React attaches an emojii reaction to a message.
// Reactions are formatted like:
//  :+1:
func (b *Bot) React(msg *slack.Msg, reaction string) {
	b.api.AddReaction(reaction, slack.NewRefToMessage(msg.Channel, msg.Timestamp)) //nolint:errcheck
}

func (b *Bot) initInfo() error {
	at, err := b.api.AuthTest()
	if err != nil {
		b.Log.Error("Unable to auth", zap.Error(err))
		return err
	}

	b.userId = at.UserID
	b.botId = at.BotID

	pageToken := ""
	for {
		channels, nextPage, err := b.api.GetConversations(&slack.GetConversationsParameters{Cursor: pageToken})
		if err != nil {
			b.Log.Error("Unable to list channels", zap.Error(err))
			return err
		}
		for _, channel := range channels {
			b.channels[channel.ID] = channel
			b.humanChannels[channel.Name] = channel
		}

		if nextPage == "" {
			break
		}
		pageToken = nextPage
	}

	users, err := b.api.GetUsers()
	if err != nil {
		b.Log.Error("Unable to list users", zap.Error(err))
		return err
	}
	for _, user := range users {
		b.users[user.ID] = user
		b.humanUsers[user.Name] = user
	}

	if v := os.Getenv("COMMIT_SHA"); v != "" {
		if c, ok := b.humanChannels["qdev"]; ok {
			b.Say(c.ID, fmt.Sprintf("I'm back. My version is %s", v))
		}
	}

	return nil
}

// handleEvents is a goroutine that handles and dispatches various events.
// These events include callbacks from Slack and custom webhooks for plugins.
func (b *Bot) handleEvents() {
	for {
		select {
		// Slash Command
		case slashCmd := <-b.cmdChannel:
			b.dispatchCommand(slashCmd)

		// Custom webhook
		case webhook := <-b.pluginWebhookChannel:
			b.dispatchWebhook(webhook)

		}
	}
}

// Start activates the Bot, creating a new API client.
// It also calls out to the Slack API to obtain all of the channels and users.
func (b *Bot) Start() {
	go b.WebhookServer()
	go b.handleEvents()
	err := b.initInfo()
	if err != nil {
		panic(err)
	}
}

// Stop cancel's the bots main context, closes the DB handle, and disconnects from slack
func (b *Bot) Stop() {
	b.cancel()
	b.wg.Wait()
	if b.db != nil {
		b.db.Close()
	}
}

// NewBot creates a new instance of Bot for use.
//
// apiKey is the Slack API key that the Bot should use to authenticate
//
// verificationToken is the webhook token that is used to validate webhooks are coming from slack
//
// dbPath is the path to the database on the filesystem.
func NewBot(parentCtx context.Context, apiKey, verificationToken, dbPath string) (*Bot, error) {
	// Seed the RNG with the current time globally
	rand.Seed(time.Now().UnixNano())

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	log, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parentCtx)

	return &Bot{
		Log:                  log,
		ctx:                  ctx,
		cancel:               cancel,
		apiKey:               apiKey,
		verificationToken:    verificationToken,
		api:                  slack.New(apiKey, slack.OptionDebug(true)),
		channels:             make(map[string]slack.Channel, 10),
		humanChannels:        make(map[string]slack.Channel),
		humanUsers:           make(map[string]slack.User),
		users:                make(map[string]slack.User),
		commands:             make(map[string]*registeredCommand),
		cmdChannel:           make(chan *slashCommand),
		webhooks:             make(map[string]*registeredWebhook),
		pluginWebhookChannel: make(chan *PluginWebhook),
		reactionHooks:        []*registeredReactionHook{},
		hooks:                []*registeredHook{},
		db:                   db,
	}, nil
}
