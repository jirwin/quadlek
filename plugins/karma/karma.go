package karma

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
)

func scoreCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			if cmdMsg.Command.Text == "" {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text: "I need a name to look up the score for.",
				}
				continue
			}
			err := cmdMsg.Store.Get(cmdMsg.Command.Text, func(val []byte) error {
				score := string(val)
				if val == nil {
					score = "0"
				}

				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      fmt.Sprintf("Score for %s is %s", cmdMsg.Command.Text, score),
					InChannel: true,
				}

				return nil
			})
			if err != nil {
				log.WithFields(log.Fields{
					"err":  err,
					"text": cmdMsg.Command.Text,
				}).Error("unable to get score")
				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text: fmt.Sprintf("Unable to fetch score for %s", cmdMsg.Command.Text),
				})
			}

		case <-ctx.Done():
			log.Info("Exiting KarmaScoreCommand.")
			return
		}
	}
}

var (
	ppRegex = regexp.MustCompile(".+\\+\\+$")
	mmRegex = regexp.MustCompile(".+--$")
)

func karmaHook(ctx context.Context, hookChannel <-chan *quadlek.HookMsg) {
	for {
		select {
		case hookMsg := <-hookChannel:
			tokens := strings.Split(hookMsg.Msg.Text, " ")

			for _, t := range tokens {
				match := ppRegex.FindString(t)
				if match != "" {
					item := match[:len(match)-2]
					err := hookMsg.Store.GetAndUpdate(item, func(val []byte) ([]byte, error) {
						if val == nil {
							return []byte("1"), nil
						}

						karma, err := strconv.Atoi(string(val[:]))
						if err != nil {
							return nil, err
						}

						karma++
						karmaStr := strconv.Itoa(karma)

						return []byte(karmaStr), nil
					})
					if err != nil {
						log.WithFields(log.Fields{
							"err": err,
						}).Errorf("Error incrementing value: %s", t)
					}
				}

				match = mmRegex.FindString(t)
				if match != "" {
					item := match[:len(match)-2]
					err := hookMsg.Store.GetAndUpdate(item, func(val []byte) ([]byte, error) {
						if val == nil {
							return []byte("-1"), nil
						}

						karma, err := strconv.Atoi(string(val[:]))
						if err != nil {
							return nil, err
						}

						karma--
						karmaStr := strconv.Itoa(karma)

						return []byte(karmaStr), nil
					})
					if err != nil {
						log.WithFields(log.Fields{
							"err": err,
						}).Errorf("Error decrementing value: %s", t)
					}
				}
			}

		case <-ctx.Done():
			log.Info("Exiting Karma Hook.")
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"karma",
		[]quadlek.Command{
			quadlek.MakeCommand("score", scoreCommand),
		},
		[]quadlek.Hook{
			quadlek.MakeHook(karmaHook),
		},
		nil,
		nil,
		nil,
	)
}
