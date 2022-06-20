package karma

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"context"

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
				zap.L().Error("unable to get score", zap.Error(err))
				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{ //nolint:errcheck
					Text: fmt.Sprintf("Unable to fetch score for %s", cmdMsg.Command.Text),
				})
			}

		case <-ctx.Done():
			zap.L().Info("Exiting KarmaScoreCommand.")
			return
		}
	}
}

var (
	ppRegex = regexp.MustCompile(`.+\+\+$`)
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
						zap.L().Error("Error incrementing value", zap.String("token", t), zap.Error(err))
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
						zap.L().Error("Error decrementing value: %s", zap.String("token", t), zap.Error(err))
					}
				}
			}

		case <-ctx.Done():
			zap.L().Info("Exiting Karma Hook.")
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
