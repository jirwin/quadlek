package gifs

import (
	"context"
	"go.uber.org/zap"

	"fmt"

	"strings"

	"net/url"

	"github.com/jirwin/quadlek/quadlek"
)

var gifs *Gifs

const (
	SaveReaction = "good-bot"
)

func gifCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			text := cmdMsg.Command.Text
			if text != "" {
				var gifUrl string
				var err error
				_ = cmdMsg.Store.Get(text, func(v []byte) error {
					if v != nil {
						cmdMsg.Command.Reply() <- &quadlek.CommandResp{
							Text:      string(v),
							InChannel: true,
						}
						return nil
					}
					gifUrl, err = gifs.Translate(text)
					if err != nil {
						cmdMsg.Command.Reply() <- &quadlek.CommandResp{
							Text:      fmt.Sprintf("an error occured: %s", err.Error()),
							InChannel: false,
						}
						return nil
					}
					cmdMsg.Command.Reply() <- &quadlek.CommandResp{
						Text:      gifUrl,
						InChannel: true,
					}
					return nil
				})
				if gifUrl != "" {
					err = cmdMsg.Store.Update(fmt.Sprintf("url:%s", gifUrl), []byte(text))
					if err != nil {
						zap.L().Error("error updating store with gif url", zap.Error(err))
					}
				}

			}

		case <-ctx.Done():
			return
		}
	}
}

func gifSaveCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			text := cmdMsg.Command.Text
			parts := strings.Split(text, " ")
			if len(parts) < 2 {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      "Malformed command: /gsave <url> phrase to save",
					InChannel: false,
				}
				continue
			}

			gUrl, err := url.Parse(parts[0])
			if err != nil {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      fmt.Sprintf("Invalid url: %s", parts[1]),
					InChannel: false,
				}
				continue
			}

			phrase := strings.Join(parts[1:], " ")

			err = cmdMsg.Store.Update(phrase, []byte(gUrl.String()))
			if err != nil {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      fmt.Sprintf("Unable to save gif phrase: %s", err.Error()),
					InChannel: false,
				}
				continue
			}

			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text:      "Successfully stored gif phrase.",
				InChannel: false,
			}

		case <-ctx.Done():
			return
		}
	}
}

func gifReaction(ctx context.Context, reactionChannel <-chan *quadlek.ReactionHookMsg) {
	for {
		select {
		case rh := <-reactionChannel:
			if strings.HasPrefix(rh.Reaction.Reaction, SaveReaction) {
				msg, err := rh.Bot.GetMessage(rh.Reaction.Item.Channel, rh.Reaction.Item.Timestamp)
				if err != nil {
					fmt.Println("error getting message:", err.Error())
					continue
				}

				if msg.User != rh.Bot.GetUserId() {
					continue
				}

				err = rh.Store.Get(fmt.Sprintf("url:%s", msg.Text), func(b []byte) error {
					if b != nil {
						return nil
					}
					err = rh.Store.Update(string(b), []byte(msg.Text))
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					rh.Bot.Say(rh.Reaction.Item.Channel, "Error saving gif")
					continue
				}

			}

		case <-ctx.Done():
			fmt.Println("Shutting down gif react hook")
			return
		}
	}
}

func Register(apiKey string) quadlek.Plugin {
	gifs = NewGifs(apiKey, "R")
	return quadlek.MakePlugin(
		"gifs",
		[]quadlek.Command{
			quadlek.MakeCommand("g", gifCommand),
			quadlek.MakeCommand("gsave", gifSaveCommand),
		},
		nil,
		[]quadlek.ReactionHook{
			quadlek.MakeReactionHook(gifReaction),
		},
		nil,
		nil,
	)
}
