package karma

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/plugin_manager"
)

func scoreCommand(ctx context.Context, cmdChannel <-chan *plugin_manager.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			if cmdMsg.Command.Text == "" {
				cmdMsg.Command.Reply() <- &plugin_manager.CommandResp{
					Text: "I need a name to look up the score for.",
				}
				continue
			}
			err := cmdMsg.Helper.Store().Get(cmdMsg.Command.Text, func(val []byte) error {
				score := string(val)
				if val == nil {
					score = "0"
				}

				cmdMsg.Command.Reply() <- &plugin_manager.CommandResp{
					Text:      fmt.Sprintf("Score for %s is %s", cmdMsg.Command.Text, score),
					InChannel: true,
				}

				return nil
			})
			if err != nil {
				zap.L().Error("unable to get score", zap.Error(err))
				cmdMsg.Helper.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &plugin_manager.CommandResp{ //nolint:errcheck
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

func karmaHook(ctx context.Context, hookChannel <-chan *plugin_manager.HookMsg) {
	for {
		select {
		case hookMsg := <-hookChannel:
			tokens := strings.Split(hookMsg.Msg.Text, " ")

			for _, t := range tokens {
				match := ppRegex.FindString(t)
				if match != "" {
					item := match[:len(match)-2]
					err := hookMsg.Helper.Store().GetAndUpdate(item, func(val []byte) ([]byte, error) {
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
					err := hookMsg.Helper.Store().GetAndUpdate(item, func(val []byte) ([]byte, error) {
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

func Register() plugin_manager.Plugin {
	return plugin_manager.MakePlugin(
		"karma",
		[]plugin_manager.Command{
			plugin_manager.MakeCommand("score", scoreCommand),
		},
		[]plugin_manager.Hook{
			plugin_manager.MakeHook(karmaHook),
		},
		nil,
		nil,
		nil,
	)
}
