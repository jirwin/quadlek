package admin

import (
	"context"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/plugin_manager"
)

func shutdown(ctx context.Context, cmdChannel <-chan *plugin_manager.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- &plugin_manager.CommandResp{
				Text: "Shutting down in 5 seconds...",
			}
			go func(helper plugin_manager.PluginHelper) {
				<-time.After(5 * time.Second)
				cmdMsg.Helper.StopBot()
			}(cmdMsg.Helper)

		case <-ctx.Done():
			zap.L().Info("Exiting quit command.")
			return
		}
	}
}

func restartInteraction(ctx context.Context, interactionChannel <-chan *plugin_manager.InteractionMsg) {
	for {
		select {
		case scMsg := <-interactionChannel:
			callbackID := ""

			switch scMsg.Interaction.Type {
			case "view_submission":
				callbackID = scMsg.Interaction.View.CallbackID
			default:
				callbackID = scMsg.Interaction.CallbackID
			}

			switch callbackID {
			case "restart":
				r := slack.ModalViewRequest{
					Type: "modal",
					Title: &slack.TextBlockObject{
						Type: "plain_text",
						Text: "Restart Quadlek",
					},
					Blocks: slack.Blocks{
						BlockSet: []slack.Block{
							&slack.SectionBlock{
								Type: "section",
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
				_, err := scMsg.Helper.OpenView(scMsg.Interaction.TriggerID, r)
				if err != nil {
					zap.L().Error("error opening view", zap.Error(err))
					continue
				}

			case "restart-confirm-modal":
				zap.L().Info("shutting down...")
				scMsg.Helper.StopBot()
			}

		case <-ctx.Done():
			zap.L().Info("Exiting quit command.")
			return
		}
	}
}

func Register() plugin_manager.Plugin {
	return plugin_manager.MakePlugin(
		"admin",
		[]plugin_manager.Command{
			plugin_manager.MakeCommand("shutdown", shutdown),
		},
		nil,
		nil,
		nil,
		nil,
	)
}

func RegisterInteraction() plugin_manager.InteractionPlugin {
	return plugin_manager.MakeInteractionPlugin(
		"restart-quadlek",
		[]plugin_manager.Interaction{
			plugin_manager.MakeInteraction("restart", restartInteraction),
		},
	)
}
