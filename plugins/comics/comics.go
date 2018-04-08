//go:generate protoc --go_out=. comics.proto

package comics

import (
	"context"

	"fmt"
	"strings"

	"math/rand"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/jirwin/comics/src/comics"
	"github.com/jirwin/quadlek/quadlek"
)

var (
	clientId string
)

func addComicTemplate(templateUrl string, cmdMsg *quadlek.CommandMsg) error {
	err := cmdMsg.Store.GetAndUpdate("templates", func(templatesProto []byte) ([]byte, error) {
		templates := &Templates{}
		err := proto.Unmarshal(templatesProto, templates)
		if err != nil {
			return nil, err
		}

		templates.Urls = append(templates.Urls, templateUrl)

		updatedTemplates, err := proto.Marshal(templates)
		if err != nil {
			return nil, err
		}

		return updatedTemplates, nil
	})

	return err
}

func listTemplates(cmdMsg *quadlek.CommandMsg) ([]string, error) {
	var templateUrls []string

	err := cmdMsg.Store.Get("templates", func(templateProto []byte) error {
		templates := &Templates{}
		err := proto.Unmarshal(templateProto, templates)
		if err != nil {
			return err
		}
		templateUrls = templates.Urls
		return nil
	})

	if err != nil {
		return nil, err
	}

	return templateUrls, nil
}

func pickAndRenderTemplate(cmdMsg *quadlek.CommandMsg) (string, error) {
	filler := []string{
		"Hi i made a joke",
		"That joke was so funny",
		"You're awesome. And I mean really awesome.",
		"Even more filler text while I test this",
	}

	comicUrl := ""

	err := cmdMsg.Store.Get("templates", func(templatesProto []byte) error {
		templates := &Templates{}
		err := proto.Unmarshal(templatesProto, templates)
		if err != nil {
			return err
		}

		if len(templates.Urls) == 0 {
			return fmt.Errorf("error: no configured templates")
		}

		template := templates.Urls[rand.Intn(len(templates.Urls))]
		comic, err := comics.NewTemplate(template)
		if err != nil {
			return err
		}

		imgBytes, err := comic.Render(filler)
		if err != nil {
			return err
		}

		comicUrl, err = comics.ImgurUpload(imgBytes, clientId)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return comicUrl, nil
}

func comicCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- nil

			if cmdMsg.Command.Text != "" {
				split := strings.Split(cmdMsg.Command.Text, " ")
				switch split[0] {
				case "list":
					templates, err := listTemplates(cmdMsg)
					if err != nil {
						cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      fmt.Sprintf("error listing template"),
							InChannel: false,
						})
						continue
					}

					if len(templates) == 0 {
						cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      "There are no templates configured.",
							InChannel: false,
						})
						continue
					}

					msgText := "Configured templates:\n"
					for i, template := range templates {
						msgText += fmt.Sprintf("%d. %s\n", i, template)
					}

					cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
						Text:      msgText,
						InChannel: false,
					})

				case "load":
					if len(split) != 2 {
						cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      "You must provide the url of a template to load.",
							InChannel: false,
						})
						continue
					}
					err := addComicTemplate(split[1], cmdMsg)
					if err != nil {
						cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      fmt.Sprintf("error adding template: %s", err.Error()),
							InChannel: false,
						})
						continue
					}

					cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
						Text:      "Successfully added template " + split[1],
						InChannel: false,
					})
				}

				continue
			}

			comicUrl, err := pickAndRenderTemplate(cmdMsg)
			if err != nil {
				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text:      fmt.Sprintf("error rendering template: %s", err.Error()),
					InChannel: false,
				})
				continue
			}

			cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
				Text:      fmt.Sprintf("<@%s> made a new comic: %s", cmdMsg.Command.UserId, comicUrl),
				InChannel: true,
			})

		case <-ctx.Done():
			log.Info("Exiting comic command.")
			return
		}
	}
}

func Register(imgurClientId string) quadlek.Plugin {
	clientId = imgurClientId

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
