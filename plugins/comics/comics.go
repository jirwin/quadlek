//go:generate protoc --go_out=. comics.proto

package comics

import (
	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
)

var (
	clientId string
)

func comicCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text:      "got a comic request",
				InChannel: false,
			}

		case <-ctx.Done():
			log.Info("Exiting comic command.")
			return
		}
	}
}

func Register(clientId string) quadlek.Plugin {
	clientId = clientId

	return quadlek.MakePlugin(
		"comics",
		[]quadlek.Command{
			quadlek.MakeCommand("comic", comicCommand),
		},
		nil,
		nil,
		nil,
		nil,
	)
}
