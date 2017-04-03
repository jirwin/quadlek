package quadlek

import (
	"fmt"

	"github.com/nlopes/slack"
)

type Bot struct {
	apiKey   string
	api      *slack.Client
	rtm      *slack.RTM
	channels map[string]slack.Channel
	username string
	userId   string
	commands map[string]Command
	hooks    []Hook
}

func (b *Bot) Respond(msg *slack.Msg, resp string) {
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(fmt.Sprintf("<@%s>: %s", msg.User, resp), msg.Channel))
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
		case msg := <-b.rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:

			case *slack.ConnectedEvent:
				b.username = ev.Info.User.Name
				b.userId = ev.Info.User.ID
				for _, channel := range ev.Info.Channels {
					if channel.IsMember {
						b.channels[channel.ID] = channel
						b.Say(channel.ID, "I'm alive!")

					}
				}

			case *slack.ChannelJoinedEvent:
				b.channels[ev.Channel.ID] = ev.Channel
				b.Say(ev.Channel.ID, "I'm alive!")

			case *slack.ChannelLeftEvent:
				delete(b.channels, ev.Channel)

			case *slack.MessageEvent:
				fmt.Println(ev.Msg.Text)
				if b.MsgToBot(ev.Msg.Text) {
					b.DispatchCommand(&ev.Msg)
				}
				if ev.Msg.User != b.userId {
					b.DispatchHooks(&ev.Msg)
				}

			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %v\n", ev)

			case *slack.LatencyReport:
				fmt.Printf("Current latency: %v\n", ev.Value)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")

			default:
				fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}

func (b *Bot) Start() {
	b.rtm = b.api.NewRTM()
	go b.rtm.ManageConnection()
	go b.HandleEvents()
}

func (b *Bot) Stop() {
	b.rtm.Disconnect()
}

func NewBot(apiKey string) *Bot {
	return &Bot{
		apiKey:   apiKey,
		api:      slack.New(apiKey),
		channels: make(map[string]slack.Channel, 10),
		commands: make(map[string]Command),
		hooks:    []Hook{},
	}
}
