//go:generate protoc --go_out=. comics.proto

package comics

import (
	"context"

	"fmt"
	"strings"

	"math/rand"

	"strconv"

	"html"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/jirwin/comics/src/comics"
	"github.com/jirwin/quadlek/quadlek"
)

var (
	clientId string
	fontPath string
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

func delComicTemplate(templateId string, cmdMsg *quadlek.CommandMsg) error {
	return cmdMsg.Store.GetAndUpdate("templates", func(templateProto []byte) ([]byte, error) {
		templates := &Templates{}
		err := proto.Unmarshal(templateProto, templates)
		if err != nil {
			return nil, err
		}
		newTemplateUrls := []string{}
		tId, err := strconv.Atoi(templateId)
		if err != nil {
			return nil, err
		}

		for i, url := range templates.Urls {
			if i != tId {
				newTemplateUrls = append(newTemplateUrls, url)
			}
		}

		templates.Urls = newTemplateUrls
		templateBytes, err := proto.Marshal(templates)
		if err != nil {
			return nil, err
		}

		return templateBytes, nil
	})
}

func formatLogMsg(text string) string {
	return html.UnescapeString(text)
}

func pickAndRenderTemplate(cmdMsg *quadlek.CommandMsg) (string, error) {
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
		comic, err := comics.NewTemplate(template, fontPath)
		if err != nil {
			return err
		}

		msgs, err := cmdMsg.Bot.GetMessageLog(cmdMsg.Command.ChannelId, quadlek.MessageLotOpts{
			SkipAttachments: true,
		})
		if err != nil {
			return err
		}

		if len(msgs) < len(comic.Bubbles) {
			return fmt.Errorf("Not enough channel history for this comic.")
		}

		comicTxt := []string{}
		for i := len(comic.Bubbles) - 1; i >= 0; i-- {
			comicTxt = append(comicTxt, formatLogMsg(msgs[i].Text))
		}

		imgBytes, err := comic.Render(comicTxt)
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

				case "del":
					if len(split) != 2 {
						cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      "You must provide the id of the template to delete.",
							InChannel: false,
						})
						continue
					}
					err := delComicTemplate(split[1], cmdMsg)
					if err != nil {
						cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      fmt.Sprintf("error deleting template: %s", err.Error()),
							InChannel: false,
						})
						continue
					}

					cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
						Text:      "Successfully deleted template " + split[1],
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

func Register(imgurClientId, comicFontPath string) quadlek.Plugin {
	clientId = imgurClientId
	fontPath = comicFontPath

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
