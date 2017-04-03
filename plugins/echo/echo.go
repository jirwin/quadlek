package echo

import (
	"github.com/jirwin/quadlek/quadlek"
	"github.com/nlopes/slack"
)

type EchoCommand struct{}

func (ec *EchoCommand) GetName() string {
	return "echo"
}

func (ec *EchoCommand) RunCommand(bot *quadlek.Bot, msg *slack.Msg, parsedMsg string, store *quadlek.Store) {
	bot.Respond(msg, parsedMsg)
}

type Plugin struct {
	Commands []quadlek.Command
}

func (p *Plugin) GetCommands() []quadlek.Command {
	return p.Commands
}

func (p *Plugin) GetHooks() []quadlek.Hook {
	return nil
}

func (p *Plugin) GetId() string {
	return "286647df-8085-48ae-936e-2190783199db"
}

func Register() quadlek.Plugin {
	return &Plugin{
		Commands: []quadlek.Command{&EchoCommand{}},
	}
}
