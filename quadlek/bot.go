package quadlek

import (
	"fmt"
	"strings"

	"errors"

	"github.com/nlopes/slack"
)

type Bot struct {
	ApiKey   string
	Api      *slack.Client
	rtm      *slack.RTM
	Channels map[string]slack.Channel
	Username string
	UserID   string
	commands map[string]Command
	hooks    []Hook
}

func (b *Bot) MsgToBot(msg string) bool {
	return strings.HasPrefix(msg, fmt.Sprintf("<@%s> ", b.UserID))
}

func (b *Bot) ParseMessage(msg string) (string, string) {
	trimmedMsg := strings.TrimPrefix(msg, fmt.Sprintf("<@%s> ", b.UserID))
	parsedMsg := strings.Split(trimmedMsg, " ")

	cmd := ""
	msgText := []string{}

	if len(parsedMsg) == 1 {
		cmd = parsedMsg[0]
	} else if len(parsedMsg) > 1 {
		cmd, msgText = parsedMsg[0], parsedMsg[1:]
	}

	return cmd, strings.Join(msgText, " ")
}

func (b *Bot) GetCommand(cmdText string) Command {
	if cmdText == "" {
		return nil
	}

	if cmd, ok := b.commands[cmdText]; ok {
		return cmd
	}

	return nil
}

func (b *Bot) RegisterPlugin(plugin Plugin) error {
	for _, command := range plugin.GetCommands() {
		_, ok := b.commands[command.GetName()]
		if ok {
			return errors.New(fmt.Sprintf("Command already exists: %s", command.GetName()))
		}
		b.commands[command.GetName()] = command
	}

	for _, hook := range plugin.GetHooks() {
		b.hooks = append(b.hooks, hook)
	}

	return nil
}

func (b *Bot) DispatchCommand(msg *slack.Msg) {
	fmt.Printf("Commands: %v", b.commands)
	cmdText, parsedMsg := b.ParseMessage(msg.Text)
	cmd := b.GetCommand(cmdText)
	if cmd == nil {
		return
	}

	go cmd.RunCommand(b, msg, parsedMsg)
}

func (b *Bot) Respond(msg *slack.Msg, resp string) {
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(fmt.Sprintf("<@%s>: %s", msg.User, resp), msg.Channel))
}

func (b *Bot) Say(channel string, resp string) {
	b.rtm.SendMessage(b.rtm.NewOutgoingMessage(fmt.Sprintf("%s", resp), channel))
}

func (b *Bot) React(msg *slack.Msg, reaction string) {
	b.Api.AddReaction(reaction, slack.NewRefToMessage(msg.Channel, msg.Timestamp))
}

func (b *Bot) HandleEvents() {
	for {
		select {
		case msg := <-b.rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:

			case *slack.ConnectedEvent:
				b.Username = ev.Info.User.Name
				b.UserID = ev.Info.User.ID
				for _, channel := range ev.Info.Channels {
					if channel.IsMember {
						b.Channels[channel.ID] = channel
						b.Say(channel.ID, "I'm alive!")

					}
				}

			case *slack.ChannelJoinedEvent:
				b.Channels[ev.Channel.ID] = ev.Channel
				b.Say(ev.Channel.ID, "I'm alive!")

			case *slack.ChannelLeftEvent:
				delete(b.Channels, ev.Channel)

			case *slack.MessageEvent:
				fmt.Println(ev.Msg.Text)
				if b.MsgToBot(ev.Msg.Text) {
					b.DispatchCommand(&ev.Msg)
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
	b.rtm = b.Api.NewRTM()
	go b.rtm.ManageConnection()
	go b.HandleEvents()
}

func (b *Bot) Stop() {
	b.rtm.Disconnect()
}

func NewBot(apiKey string) *Bot {
	return &Bot{
		ApiKey:   apiKey,
		Api:      slack.New(apiKey),
		Channels: make(map[string]slack.Channel, 10),
		commands: make(map[string]Command),
		hooks:    []Hook{},
	}
}
