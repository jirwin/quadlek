package echo

import (
	"context"

	"go.uber.org/zap"

	"fmt"

	"github.com/jirwin/quadlek/quadlek"
)

func echoCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text: cmdMsg.Command.Text,
			}
		case <-ctx.Done():
			zap.L().Info("Exiting echo command")
			return
		}
	}
}

func echoReactionHook(ctx context.Context, reactionChannel <-chan *quadlek.ReactionHookMsg) {
	for {
		select {
		case rh := <-reactionChannel:
			user, err := rh.Bot.GetUserName(rh.Reaction.User)
			if err != nil {
				zap.L().Error("User not found.", zap.Error(err))
				continue
			}
			rh.Bot.Say(rh.Reaction.Item.Channel, fmt.Sprintf("@%s added a reaction! :%s:", user, rh.Reaction.Reaction))

		case <-ctx.Done():
			zap.L().Info("Exiting echo reaction hook")
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"echo",
		[]quadlek.Command{quadlek.MakeCommand("echo", echoCommand)},
		nil,
		nil,
		nil,
		nil,
	)
}
