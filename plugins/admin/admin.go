package admin

import (
	"context"
	"github.com/jirwin/quadlek/quadlek"
	"go.uber.org/zap"
)

func shutdown(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text: "Shutting down...",
			}
			cmdMsg.Bot.Stop()

		case <-ctx.Done():
			zap.L().Info("Exiting quit command.")
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"admin",
		[]quadlek.Command{
			quadlek.MakeCommand("shutdown", shutdown),
		},
		nil,
		nil,
		nil,
		nil,
	)
}
