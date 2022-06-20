package comics

import (
	"context"
	"fmt"
	v1 "github.com/jirwin/quadlek/pb/quadlek/plugins/comics/v1"
	"html"
	"math/rand"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/jirwin/comics/src/comics"
	"github.com/jirwin/quadlek/quadlek"
)

var (
	clientId string
	fontPath string
)

func addComicTemplate(templateUrl string, cmdMsg *quadlek.CommandMsg) error {
	err := cmdMsg.Store.GetAndUpdate("templates", func(templatesProto []byte) ([]byte, error) {
		templates := &v1.Templates{}
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
		templates := &v1.Templates{}
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
		templates := &v1.Templates{}
		err := proto.Unmarshal(templateProto, templates)
		if err != nil {
			return nil, err
		}
		var newTemplateUrls []string
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
		templates := &v1.Templates{}
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
			return fmt.Errorf("not enough channel history for this comic")
		}

		var comicTxt []string
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
						err := cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      "error listing template",
							InChannel: false,
						})
						if err != nil {
							return
						}
						continue
					}

					if len(templates) == 0 {
						err := cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      "There are no templates configured.",
							InChannel: false,
						})
						if err != nil {
							return
						}
						continue
					}

					msgText := "Configured templates:\n"
					for i, template := range templates {
						msgText += fmt.Sprintf("%d. %s\n", i, template)
					}

					err = cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
						Text:      msgText,
						InChannel: false,
					})
					if err != nil {
						return
					}

				case "del":
					if len(split) != 2 {
						err := cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      "You must provide the id of the template to delete.",
							InChannel: false,
						})
						if err != nil {
							return
						}
						continue
					}
					err := delComicTemplate(split[1], cmdMsg)
					if err != nil {
						err := cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      fmt.Sprintf("error deleting template: %s", err.Error()),
							InChannel: false,
						})
						if err != nil {
							return
						}
						continue
					}

					err = cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
						Text:      "Successfully deleted template " + split[1],
						InChannel: false,
					})
					if err != nil {
						return
					}

				case "load":
					if len(split) != 2 {
						err := cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      "You must provide the url of a template to load.",
							InChannel: false,
						})
						if err != nil {
							return
						}
						continue
					}
					err := addComicTemplate(split[1], cmdMsg)
					if err != nil {
						err := cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
							Text:      fmt.Sprintf("error adding template: %s", err.Error()),
							InChannel: false,
						})
						if err != nil {
							return
						}
						continue
					}

					err = cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
						Text:      "Successfully added template " + split[1],
						InChannel: false,
					})
					if err != nil {
						return
					}
				}

				continue
			}

			comicUrl, err := pickAndRenderTemplate(cmdMsg)
			if err != nil {
				err := cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text:      fmt.Sprintf("error rendering template: %s", err.Error()),
					InChannel: false,
				})
				if err != nil {
					return
				}
				continue
			}

			err = cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
				Text:      fmt.Sprintf("<@%s> made a new comic: %s", cmdMsg.Command.UserId, comicUrl),
				InChannel: true,
			})
			if err != nil {
				return
			}

		case <-ctx.Done():
			zap.L().Info("Exiting comic command.")
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
