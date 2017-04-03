package echo

import (
	"github.com/jirwin/quadlek/quadlek"
	"github.com/nlopes/slack"
)

type EchoCommand struct{}

func (ec *EchoCommand) GetName() string {
	return "echo"
}

func (ec *EchoCommand) RunCommand(bot *quadlek.Bot, msg *slack.Msg, parsedMsg string) {
	bot.Respond(msg, parsedMsg)
}

type Plugin struct {
	Commands []quadlek.Command
	Hooks    []quadlek.Hook
}

func (p Plugin) GetCommands() []quadlek.Command {
	return p.Commands
}

func (p Plugin) GetHooks() []quadlek.Hook {
	return p.Hooks
}

func Register() quadlek.Plugin {
	return &Plugin{
		Commands: []quadlek.Command{&EchoCommand{}},
	}
}
