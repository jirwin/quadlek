package quadlek

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nlopes/slack"
)

type Command interface {
	GetName() string
	RunCommand(bot *Bot, msg *slack.Msg, parsedMsg string)
}

type Hook interface {
	RunHook(bot *Bot, msg *slack.Msg)
}

type Plugin interface {
	GetCommands() []Command
	GetHooks() []Hook
}

func (b *Bot) MsgToBot(msg string) bool {
	return strings.HasPrefix(msg, fmt.Sprintf("<@%s> ", b.userId))
}

func (b *Bot) ParseMessage(msg string) (string, string) {
	trimmedMsg := strings.TrimPrefix(msg, fmt.Sprintf("<@%s> ", b.userId))
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
	cmdText, parsedMsg := b.ParseMessage(msg.Text)
	cmd := b.GetCommand(cmdText)
	if cmd == nil {
		return
	}

	go cmd.RunCommand(b, msg, parsedMsg)
}

func (b *Bot) DispatchHooks(msg *slack.Msg) {
	for _, hook := range b.hooks {
		go hook.RunHook(b, msg)
	}
}
