package lib

import (
	"fmt"
	"github.com/nlopes/slack"
	"strings"
)

type Bot struct {
	ApiKey   string
	Api      *slack.Client
	Channels map[string]slack.Channel
	Username string
	UserID   string
}

func (b *Bot) MsgToBot(msg string) bool {
	return strings.HasPrefix(msg, fmt.Sprintf("<@%s>:", b.UserID))
}

func (b *Bot) ParseMessage(msg string) []string {
	parsedMsg := strings.TrimPrefix(msg, fmt.Sprintf("<@%s>:", b.UserID))
	return strings.Split(parsedMsg, " ")
}

func (b *Bot) HandleEvents(rtm *slack.RTM) {
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			fmt.Print("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:

			case *slack.ConnectedEvent:
				fmt.Println("Infos:", ev.Info.Channels)
				for _, channel := range ev.Info.Channels {
					if channel.IsMember {
						b.Channels[channel.ID] = channel
						rtm.SendMessage(rtm.NewOutgoingMessage("I'm Alive!", channel.ID))

					}
				}
				fmt.Println("Connection counter:", ev.ConnectionCount)

			case *slack.ChannelJoinedEvent:
				fmt.Printf("Joining channel: %s\n", ev.Channel.Name)
				b.Channels[ev.Channel.ID] = ev.Channel
				rtm.SendMessage(rtm.NewOutgoingMessage("Hi!", ev.Channel.ID))

			case *slack.ChannelLeftEvent:
				fmt.Printf("Leaving channel: %s\n", ev.Channel)
				delete(b.Channels, ev.Channel)

			case *slack.MessageEvent:
				fmt.Printf("Message: %v\n", ev)

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

func (b *Bot) StartRTM() {
	rtm := b.Api.NewRTM()
	go rtm.ManageConnection()
	go b.HandleEvents(rtm)
}

func NewBot(apiKey string) *Bot {
	return &Bot{
		ApiKey:   apiKey,
		Api:      slack.New(apiKey),
		Channels: make(map[string]slack.Channel, 10),
	}
}
