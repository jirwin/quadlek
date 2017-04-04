package quadlek

import (
	"errors"
	"fmt"
	"strings"

	"context"

	"github.com/nlopes/slack"
)

type Command interface {
	GetName() string
	Channel() chan<- *CommandMsg
	Run(ctx context.Context)
}

type registeredCommand struct {
	PluginId string
	Command  Command
}

type Hook interface {
	Channel() chan<- *HookMsg
	Run(ctx context.Context)
}

type CommandMsg struct {
	Bot       *Bot
	Msg       *slack.Msg
	ParsedMsg string
	Store     *Store
}

type HookMsg struct {
	Bot   *Bot
	Msg   *slack.Msg
	Store *Store
}

type registeredHook struct {
	PluginId string
	Hook     Hook
}

type Plugin interface {
	GetId() string
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

func (b *Bot) GetCommand(cmdText string) *registeredCommand {
	if cmdText == "" {
		return nil
	}

	if cmd, ok := b.commands[cmdText]; ok {
		return cmd
	}

	return nil
}

func (b *Bot) RegisterPlugin(plugin Plugin) error {
	if plugin.GetId() == "" {
		return errors.New("Must provide a unique plugin id.")
	}

	err := b.InitPluginBucket(plugin.GetId())
	if err != nil {
		return err
	}

	for _, command := range plugin.GetCommands() {
		_, ok := b.commands[command.GetName()]
		if ok {
			return errors.New(fmt.Sprintf("Command already exists: %s", command.GetName()))
		}
		b.commands[command.GetName()] = &registeredCommand{
			PluginId: plugin.GetId(),
			Command:  command,
		}
		go func() {
			b.wg.Add(1)
			defer b.wg.Done()

			command.Run(b.ctx)
		}()
	}

	for _, hook := range plugin.GetHooks() {
		b.hooks = append(b.hooks, &registeredHook{
			PluginId: plugin.GetId(),
			Hook:     hook,
		})
		go func() {
			b.wg.Add(1)
			defer b.wg.Done()

			hook.Run(b.ctx)
		}()
	}

	return nil
}

func (b *Bot) DispatchCommand(msg *slack.Msg) {
	cmdText, parsedMsg := b.ParseMessage(msg.Text)
	cmd := b.GetCommand(cmdText)
	if cmd == nil {
		return
	}

	store := &Store{
		db:       b.db,
		pluginId: cmd.PluginId,
	}

	cmd.Command.Channel() <- &CommandMsg{
		Bot:       b,
		Msg:       msg,
		ParsedMsg: parsedMsg,
		Store:     store,
	}
}

func (b *Bot) DispatchHooks(msg *slack.Msg) {
	for _, hook := range b.hooks {
		store := &Store{
			db:       b.db,
			pluginId: hook.PluginId,
		}

		hook.Hook.Channel() <- &HookMsg{
			Bot:   b,
			Msg:   msg,
			Store: store,
		}
	}
}
