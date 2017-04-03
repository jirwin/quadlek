package echo

import (
	"github.com/jirwin/quadlek/lib"
	"strings"
)

type EchoCommand struct {}

func (ec *EchoCommand) GetName() (string) {
	return "echo"
}

func (ec *EchoCommand) RunCommand(bot *lib.Bot, from string, to string, msg []string) {
	bot.Rtm.SendMessage(bot.Rtm.NewOutgoingMessage(strings.Join(msg, " "), to))
}

type Plugin struct {
	Commands []lib.Command
	Hooks []lib.Hook
}

func (p *Plugin) GetCommands() ([]lib.Command) {
	return p.Commands
}

func (p *Plugin) RunCommands(bot *lib.Bot, from string, to string, msg []string) {
	for _, c := range p.Commands {
		go c.RunCommand(bot, from, to, msg)
	}
}

func (p *Plugin) RunHooks(bot *lib.Bot, from string, to string, msg []string) {
	for _, h := range p.Hooks {
		go h.RunHook(bot, from, to, msg)
	}
}

func Register() (*Plugin) {
	return &Plugin{
		Commands: []lib.Command{&EchoCommand{}},
		Hooks: nil,
	}
}