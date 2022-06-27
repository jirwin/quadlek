package eslogs

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/olivere/elastic.v5"

	"github.com/jirwin/quadlek/pkg/plugin_manager"
)

var (
	esEndpoint = ""
	esIndex    = ""
	esClient   *elastic.Client
)

type SlackMsgLog struct {
	Timestamp string `json:"ts"`
	Channel   string `json:"channel"`
	User      string `json:"user"`
	Text      string `json:"text"`
}

var SlackUserMatch = regexp.MustCompile("<@U.+>")

func formatText(helper plugin_manager.PluginHelper, txt string) string {
	formattedText := SlackUserMatch.ReplaceAllStringFunc(txt, func(s string) string {
		userId := strings.TrimLeft(strings.TrimRight(s, ">"), "<@")

		user, err := helper.GetUser(userId)
		if err != nil {
			return s
		}

		return user.Name
	})

	return formattedText
}

func logHook(ctx context.Context, hookchan <-chan *plugin_manager.HookMsg) {
	for {
		select {
		case hookMsg := <-hookchan:
			msg := SlackMsgLog{
				Timestamp: hookMsg.Msg.Timestamp,
			}
			channel, err := hookMsg.Helper.GetChannel(hookMsg.Msg.Channel)
			if err != nil {
				msg.Channel = "unknown"
			} else {
				msg.Channel = channel.Name
			}

			user, err := hookMsg.Helper.GetUser(hookMsg.Msg.User)
			if err != nil {
				msg.User = "unknown"
			} else {
				msg.User = user.Name
			}

			txt := formatText(hookMsg.Helper, hookMsg.Msg.Text)
			msg.Text = txt

			if hookMsg.Msg.SubType != "bot_msg" && hookMsg.Msg.SubType != "" {
				continue
			}

			_, err = esClient.Index().Index(esIndex).Type("slack-msg").Id(hookMsg.Msg.Timestamp).BodyJson(msg).Do(ctx)
			if err != nil {
				zap.L().Error("Error indexing log to ES", zap.Error(err))
				continue
			}

		case <-ctx.Done():
			zap.L().Info("Exiting es log hook")
			return
		}
	}
}

func Register(endpoint, index string) (plugin_manager.Plugin, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("es endpoint is required")
	}
	esEndpoint = endpoint

	if index == "" {
		return nil, fmt.Errorf("es index is required")
	}
	esIndex = index

	esc, err := elastic.NewClient(elastic.SetURL(esEndpoint), elastic.SetSniff(false))
	if err != nil {
		return nil, err
	}
	esClient = esc

	return plugin_manager.MakePlugin(
		"eslogs",
		nil,
		[]plugin_manager.Hook{
			plugin_manager.MakeHook(logHook),
		},
		nil,
		nil,
		nil,
	), nil
}
