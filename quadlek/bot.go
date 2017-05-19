package quadlek

import (
	"fmt"
	"time"

	"context"

	"sync"

	"math/rand"

	"errors"

	"github.com/boltdb/bolt"
	"github.com/nlopes/slack"

	log "github.com/Sirupsen/logrus"
)

type Bot struct {
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

func (b *Bot) GetUserId() string {
	return b.userId
}

func (b *Bot) GetApi() *slack.Client {
	return b.api
}

func (b *Bot) GetChannelId(chanName string) (string, error) {
	channel, ok := b.humanChannels[chanName]
	if !ok {
		return "", errors.New("Channel not found.")
	}

	return channel.ID, nil
}

func (b *Bot) GetUserName(userId string) (string, error) {
	user, ok := b.users[userId]
	if !ok {
		return "", errors.New("User not found.")
	}

	return user.Name, nil
}

func (b *Bot) Respond(msg *slack.Msg, resp string) {
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(fmt.Sprintf("<@%s>: %s", msg.User, resp), msg.Channel))
}

func (b *Bot) PostMessage(channel, resp string, params slack.PostMessageParameters) (string, string, error) {
	return b.rtm.PostMessage(channel, resp, params)
}

func (b *Bot) Say(channel string, resp string) {
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(fmt.Sprintf("%s", resp), channel))
}

func (b *Bot) React(msg *slack.Msg, reaction string) {
	b.api.AddReaction(reaction, slack.NewRefToMessage(msg.Channel, msg.Timestamp))
}

func (b *Bot) HandleEvents() {
	for {
		select {
		case slashCmd := <-b.cmdChannel:
			b.dispatchCommand(slashCmd)

		case webhook := <-b.pluginWebhookChannel:
			b.dispatchWebhook(webhook)

		case msg := <-b.rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:

			case *slack.ConnectedEvent:
				b.username = ev.Info.User.Name
				b.userId = ev.Info.User.ID

			case *slack.ChannelJoinedEvent:
				b.channels[ev.Channel.ID] = ev.Channel
				b.Say(ev.Channel.ID, "I'm alive!")

			case *slack.ChannelLeftEvent:
				delete(b.channels, ev.Channel)

			case *slack.MessageEvent:
				if ev.Msg.User != b.userId {
					b.dispatchHooks(&ev.Msg)
				}

			case *slack.ChannelCreatedEvent:
				if ev.Channel.IsChannel {
					channel, err := b.api.GetChannelInfo(ev.Channel.ID)
					if err != nil {
						log.WithError(err).Error("Unable to add channel")
						continue
					}
					b.humanChannels[channel.Name] = *channel
				}

			case *slack.UserChangeEvent:
				b.users[ev.User.ID] = ev.User
				b.humanUsers[ev.User.Name] = ev.User

			case *slack.ReactionAddedEvent:
				if ev.User != b.userId {
					b.dispatchReactions(ev)
				}

			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %v\n", ev)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")

			}
		}
	}
}

func (b *Bot) Start() {
	b.rtm = b.api.NewRTM()
	go b.rtm.ManageConnection()
	go b.HandleEvents()
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

func (b *Bot) Stop() {
	b.cancel()
	b.wg.Wait()
	if b.db != nil {
		b.db.Close()
	}
	b.rtm.Disconnect()
}

func NewBot(parentCtx context.Context, apiKey, verificationToken, dbPath string) (*Bot, error) {
	// Seed the RNG with the current time globally
	rand.Seed(time.Now().UnixNano())

	ctx, cancel := context.WithCancel(parentCtx)

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	return &Bot{
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
