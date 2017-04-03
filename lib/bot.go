package lib

import (
	"fmt"
	"github.com/nlopes/slack"
	"strings"
	"github.com/jirwin/quadlek/plugins"
)

type Bot struct {
	ApiKey   string
	Api      *slack.Client
	Rtm	*slack.RTM
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

func (b *Bot) RegisterPlugin(p *Plugin) {}

func (b *Bot) Respond(rtm *slack.RTM, msg slack.Msg, resp string) {
	rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("<@%s>: %s", msg.User, resp), msg.Channel))
}

func (b *Bot) Say(rtm *slack.RTM, channel string, resp string) {
	rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("%s", resp), channel))
}

func (b *Bot) React(msg slack.Msg, reaction string) {
	b.Api.AddReaction(reaction, slack.NewRefToMessage(msg.Channel, msg.Timestamp))
}

func (b *Bot) HandleEvents() {
	for {
		select {
		case msg := <-b.Rtm.IncomingEvents:
			fmt.Print("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:

			case *slack.ConnectedEvent:
				fmt.Println("Infos:", ev.Info.Channels)
				b.Username = ev.Info.User.Name
				b.UserID = ev.Info.User.ID
				for _, channel := range ev.Info.Channels {
					if channel.IsMember {
						b.Channels[channel.ID] = channel
						b.Say(b.Rtm, channel.ID, "I'm alive!")

					}
				}
				fmt.Println("Connection counter:", ev.ConnectionCount)

			case *slack.ChannelJoinedEvent:
				fmt.Printf("Joining channel: %s\n", ev.Channel.Name)
				b.Channels[ev.Channel.ID] = ev.Channel
				b.Say(b.Rtm, ev.Channel.ID, "I'm alive!")

			case *slack.ChannelLeftEvent:
				fmt.Printf("Leaving channel: %s\n", ev.Channel)
				delete(b.Channels, ev.Channel)

			case *slack.MessageEvent:
				fmt.Println(ev.Msg.Text)
				if (b.MsgToBot(ev.Msg.Text)) {
					b.React(ev.Msg, "partyparrot")
					b.Respond(b.Rtm, ev.Msg, "hi to you!")
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

func (b *Bot) StartRTM() {
	b.Rtm = b.Api.NewRTM()
	go b.Rtm.ManageConnection()
	go b.HandleEvents()
}

func NewBot(apiKey string) *Bot {
	return &Bot{
		ApiKey:   apiKey,
		Api:      slack.New(apiKey),
		Channels: make(map[string]slack.Channel, 10),
	}
}
