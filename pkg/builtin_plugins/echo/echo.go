package echo

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/plugin_manager"
)

func echoCommand(ctx context.Context, cmdChannel <-chan *plugin_manager.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- &plugin_manager.CommandResp{
				Text: cmdMsg.Command.Text,
			}
		case <-ctx.Done():
			zap.L().Info("Exiting echo command")
			return
		}
	}
}

func echoHook(ctx context.Context, hookChannel <-chan *plugin_manager.HookMsg) {
	for {
		select {
		case hookMsg := <-hookChannel:
			hookMsg.Helper.Respond(hookMsg.Msg, fmt.Sprintf("echo: %s", hookMsg.Msg.Text))
		case <-ctx.Done():
			zap.L().Info("Exiting echo hook")
			return
		}
	}
}

func echoReactionHook(ctx context.Context, reactionChannel <-chan *plugin_manager.ReactionHookMsg) {
	for {
		select {
		case rh := <-reactionChannel:
			rh.Helper.Say(rh.Reaction.Item.Channel, fmt.Sprintf("<@%s> added a reaction! :%s:", rh.Reaction.User, rh.Reaction.Reaction))

		case <-ctx.Done():
			zap.L().Info("Exiting echo reaction hook")
			return
		}
	}
}

func Register() plugin_manager.Plugin {
	return plugin_manager.MakePlugin(
		"echo",
		[]plugin_manager.Command{plugin_manager.MakeCommand("echo", echoCommand)},
		[]plugin_manager.Hook{plugin_manager.MakeHook(echoHook)},
		[]plugin_manager.ReactionHook{plugin_manager.MakeReactionHook(echoReactionHook)},
		nil,
		nil,
	)
}
