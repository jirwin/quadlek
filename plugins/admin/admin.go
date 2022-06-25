package admin

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/jirwin/quadlek/quadlek"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

func shutdown(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text: "Shutting down...",
			}
			cmdMsg.Bot.Stop()

		case <-ctx.Done():
			zap.L().Info("Exiting quit command.")
			return
		}
	}
}

func restartInteraction(ctx context.Context, shortcutChannel <-chan *quadlek.InteractionMsg) {
	for {
		select {
		case scMsg := <-shortcutChannel:
			r := slack.ModalViewRequest{
				Type: "modal",
				Title: &slack.TextBlockObject{
					Type: "plain_text",
					Text: "Restart Quadlek",
				},
				Blocks: slack.Blocks{
					BlockSet: []slack.Block{
						&slack.SectionBlock{
							Type: "plain_text",
							Text: &slack.TextBlockObject{
								Type: "plain_text",
								Text: "Are you sure you'd like to restart quadlek?",
							},
						},
					},
				},
				Submit: &slack.TextBlockObject{
					Type: "plain_text",
					Text: "Confirm",
				},
				CallbackID: "restart-confirm-modal",
			}
			resp, err := scMsg.Bot.OpenView(scMsg.Interaction.TriggerID, r)
			if err != nil {
				zap.L().Error("error opening view", zap.Error(err))
				continue
			}
			spew.Dump("***view response", resp)

		case <-ctx.Done():
			zap.L().Info("Exiting quit command.")
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"admin",
		[]quadlek.Command{
			quadlek.MakeCommand("shutdown", shutdown),
		},
		nil,
		nil,
		nil,
		nil,
	)
}

func RegisterInteraction() quadlek.InteractionPlugin {
	return quadlek.MakeInteractionPlugin(
		"restart-quadlek",
		[]quadlek.Interaction{
			quadlek.MakeInteraction("restart", restartInteraction),
		},
	)
}
