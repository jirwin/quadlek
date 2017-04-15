package echo

import (
	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
)

func echoCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text: cmdMsg.Command.Text,
			}
		case <-ctx.Done():
			log.Info("Exiting echo command")
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"echo",
		[]quadlek.Command{quadlek.MakeCommand("echo", echoCommand)},
		nil,
		nil,
	)
}
