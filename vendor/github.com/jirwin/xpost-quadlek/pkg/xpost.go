package xpost

import (
	"context"
	"fmt"
	"strings"

	"github.com/jirwin/quadlek/quadlek"
)

const XpostPrefix = "xpost-"

func xpostReaction(ctx context.Context, reactionChannel <-chan *quadlek.ReactionHookMsg) {
	for {
		select {
		case rh := <-reactionChannel:
			if strings.HasPrefix(rh.Reaction.Reaction, XpostPrefix) {
				dstChan := strings.TrimPrefix(rh.Reaction.Reaction, XpostPrefix)
				dstChanId, err := rh.Bot.GetChannelId(dstChan)
				if err != nil {
					fmt.Println("error getting channel id", err.Error())
					continue
				}

				if dstChanId == rh.Reaction.Item.Channel {
					continue
				}

				msg, err := rh.Bot.GetMessage(rh.Reaction.Item.Channel, rh.Reaction.Item.Timestamp)
				if err != nil {
					fmt.Println("error getting message:", err.Error())
					continue
				}
				rh.Bot.Say(dstChanId, msg.Text)
			}

		case <-ctx.Done():
			fmt.Println("Shutting down xpost")
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"xpost",
		nil,
		nil,
		[]quadlek.ReactionHook{
			quadlek.MakeReactionHook(xpostReaction),
		},
		nil,
		nil,
	)
}
