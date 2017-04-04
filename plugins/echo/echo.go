package echo

import (
	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
)

type EchoCommand struct {
	channel chan *quadlek.CommandMsg
}

func (ec *EchoCommand) GetName() string {
	return "echo"
}

func (ec *EchoCommand) Channel() chan<- *quadlek.CommandMsg {
	return ec.channel
}

func (ec *EchoCommand) Run(ctx context.Context) {
	for {
		select {
		case cmdMsg := <-ec.channel:
			cmdMsg.Bot.Respond(cmdMsg.Msg, cmdMsg.ParsedMsg)

		case <-ctx.Done():
			log.Info("Exiting echo command")
			return
		}
	}
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
		Commands: []quadlek.Command{&EchoCommand{
			channel: make(chan *quadlek.CommandMsg),
		}},
	}
}
