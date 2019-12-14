// quadlek is a slack Bot that is built on top of the nlopes Slack client.
//
// For a good source of examples, look at the included plugins at https://github.com/jirwin/quadlek/tree/master/plugins.
//
// Read more about the client and Slack APIs at: https://github.com/nlopes/slack and https://api.slack.com
package quadlek

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"context"

	"sync"

	"math/rand"

	"errors"

	"github.com/boltdb/bolt"
	"github.com/nlopes/slack"
)

// This is the core struct for the Bot, and provides all methods required for interacting with various Slack APIs.
//
// An instance of the bot is provided to plugins to enable plugins to interact with the Slack API.
type Bot struct {
	Log                  *zap.Logger
	apiKey               string
	verificationToken    string
	api                  *slack.Client
	rtm                  *slack.RTM
	channels             map[string]slack.Channel
	humanChannels        map[string]slack.Channel
	username             string
	userId               string
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

// Respond responds to a slack message
// The sent message will go to the same channel as the message that is being responded to and will highlight
// the author of the original message.
func (b *Bot) Respond(msg *slack.Msg, resp string) {
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(fmt.Sprintf("<@%s>: %s", msg.User, resp), msg.Channel))
}

// PostMessage sends a new message to Slack using the provided channel and message string.
// It returns the channel ID the message was posted to, and the timestamp that the message was posted at.
// In combination these can be used to identify the exact message that was sent.
func (b *Bot) PostMessage(channel, resp string, params slack.PostMessageParameters) (string, string, error) {
	return b.rtm.PostMessage(channel, resp, params)
}

// Say sends a new "action" on behalf of the Bot to a channel.
// This is equivalent to using
//  /me loves quadlek
func (b *Bot) Say(channel string, resp string) {
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(fmt.Sprintf("%s", resp), channel))
}

// React attaches an emojii reaction to a message.
// Reactions are formatted like:
//  :+1:
func (b *Bot) React(msg *slack.Msg, reaction string) {
	b.api.AddReaction(reaction, slack.NewRefToMessage(msg.Channel, msg.Timestamp))
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

		// Slack message
		case msg := <-b.rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:

			// This fires when the Bot connects to slack.
			// We use this opportunity to grab all of the channels and users, and index them for use later.
			case *slack.ConnectedEvent:
				b.username = ev.Info.User.Name
				b.userId = ev.Info.User.ID
				channels, err := b.api.GetChannels(true)
				if err != nil {
					b.Log.Error("Unable to list channels", zap.Error(err))
					continue
				}
				for _, channel := range channels {
					b.channels[channel.ID] = channel
					b.humanChannels[channel.Name] = channel
				}

				users, err := b.api.GetUsers()
				if err != nil {
					b.Log.Error("Unable to list users", zap.Error(err))
					continue
				}
				for _, user := range users {
					b.users[user.ID] = user
					b.humanUsers[user.Name] = user
				}

			// This fires when the Bot joins a channel
			case *slack.ChannelJoinedEvent:
				b.channels[ev.Channel.ID] = ev.Channel
				b.Say(ev.Channel.ID, "I'm alive!")

			// This fires when the Bot leaves a channel
			case *slack.ChannelLeftEvent:
				delete(b.channels, ev.Channel)

			// This fires whenever the Bot sees a message sent to slack.
			// The received message is dispatched to all plugin hooks unless the message originated from the Bot.
			case *slack.MessageEvent:
				if ev.Msg.User != b.userId {
					b.dispatchHooks(&ev.Msg)
				}

			// This fires when a new channel is created. We index the details of the new channel.
			case *slack.ChannelCreatedEvent:
				if ev.Channel.IsChannel {
					channel, err := b.api.GetChannelInfo(ev.Channel.ID)
					if err != nil {
						b.Log.Error("Unable to add channel", zap.Error(err))
						continue
					}
					b.humanChannels[channel.Name] = *channel
				}

			// This fires when a user changes. We update our index of users.
			case *slack.UserChangeEvent:
				b.users[ev.User.ID] = ev.User
				b.humanUsers[ev.User.Name] = ev.User

			// This fires whenever a reaction is attached to a message. We dispatch this event to any reaction hooks.
			case *slack.ReactionAddedEvent:
				if ev.User != b.userId {
					b.dispatchReactions(ev)
				}

			// This fires whenever a user goes away or comes back.
			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %v\n", ev)

			// This fires whenever there is an error received from the RTM API.
			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			// This fires if the Bot's credentials are invalid.
			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")

			}
		}
	}
}

// Start activates the Bot, creating a new API client and real time messaging(https://api.slack.com/rtm) client.
// It also calls out to the Slack API to obtain all of the channels and users.
func (b *Bot) Start() {
	b.rtm = b.api.NewRTM()
	go b.rtm.ManageConnection()
	go b.handleEvents()
	go b.WebhookServer()

	channels, err := b.api.GetChannels(false)
	if err != nil {
		panic(err)
	}
	for _, c := range channels {
		b.humanChannels[c.Name] = c
	}

	users, err := b.api.GetUsers()
	if err != nil {
		panic(err)
	}

	for _, u := range users {
		b.users[u.ID] = u
		b.humanUsers[u.Name] = u
	}
}

// Stop cancel's the bots main context, closes the DB handle, and disconnects from slack
func (b *Bot) Stop() {
	b.cancel()
	b.wg.Wait()
	if b.db != nil {
		b.db.Close()
	}
	b.rtm.Disconnect()
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

	ctx, cancel := context.WithCancel(parentCtx)

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	log, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	return &Bot{
		Log:                  log,
		ctx:                  ctx,
		cancel:               cancel,
		apiKey:               apiKey,
		verificationToken:    verificationToken,
		api:                  slack.New(apiKey),
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
