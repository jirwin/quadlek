//go:generate protoc --go_out=. comics.proto

package comics

import (
	"context"

	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/jirwin/quadlek/quadlek"
)

var (
	clientId string
)

func comicCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- nil

			if cmdMsg.Command.Text != "" {
				split := strings.Split(cmdMsg.Command.Text, " ")
				switch split[0] {
				case "load":
					if len(split) != 2 {
						cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      "You must provide the url of a template to load.",
							InChannel: false,
						})
					}
					err := cmdMsg.Store.GetAndUpdate("template", func(templatesProto []byte) ([]byte, error) {
						templates := &Templates{}
						err := proto.Unmarshal(templatesProto, templates)
						if err != nil {
							return nil, err
						}

						templates.Urls = append(templates.Urls, split[1])

						updatedTemplates, err := proto.Marshal(templates)
						if err != nil {
							return nil, err
						}

						return updatedTemplates, nil
					})
					if err != nil {
						cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      fmt.Sprintf("error adding template: %s", err.Error()),
							InChannel: false,
						})
					}
				}
			}
			cmdMsg.Command.Reply() <- nil

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
