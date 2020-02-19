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

func echoHook(ctx context.Context, hookChannel <-chan *quadlek.HookMsg) {
	for {
		select {
		case hookMsg := <-hookChannel:
			hookMsg.Bot.Respond(hookMsg.Msg, fmt.Sprintf("echo: %s", hookMsg.Msg.Text))
		}
	}
}

func echoReactionHook(ctx context.Context, reactionChannel <-chan *quadlek.ReactionHookMsg) {
	for {
		select {
		case rh := <-reactionChannel:
			rh.Bot.Say(rh.Reaction.Item.Channel, fmt.Sprintf("<@%s> added a reaction! :%s:", rh.Reaction.User, rh.Reaction.Reaction))

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
		[]quadlek.Hook{quadlek.MakeHook(echoHook)},
		[]quadlek.ReactionHook{quadlek.MakeReactionHook(echoReactionHook)},
		nil,
		nil,
	)
}
