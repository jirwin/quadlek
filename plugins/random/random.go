package random

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/jirwin/quadlek/quadlek"
)

func rollCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			max := int64(100)
			text := cmdMsg.Command.Text
			if text != "" {
				parsedMax, err := strconv.Atoi(text)
				if err != nil {
					cmdMsg.Command.Reply() <- &quadlek.CommandResp{
						Text: fmt.Sprintf("Sorry '%s' isn't a valid number.", text),
					}
					continue
				}

				max = int64(parsedMax)
			}

			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text:      fmt.Sprintf("You rolled a %s!", strconv.FormatInt(rand.Int63n(max+1), 10)),
				InChannel: true,
			}

		case <-ctx.Done():
			return
		}
	}
}

func chooseCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			text := cmdMsg.Command.Text
			choices := strings.Split(text, ",")

			if len(choices) == 0 {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text: "I can't make a choice for you if you don't give me any choices!",
				}
				continue
			}

			for i, choice := range choices {
				choices[i] = strings.TrimSpace(choice)
			}

			if len(choices) == 1 {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      fmt.Sprintf("Well I guess I *have* to choose %s.", choices[0]),
					InChannel: true,
				}
				continue
			}

			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text:      fmt.Sprintf("I choose %s!", choices[rand.Intn(len(choices))]),
				InChannel: true,
			}

		case <-ctx.Done():
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"random",
		[]quadlek.Command{
			quadlek.MakeCommand("roll", rollCommand),
			quadlek.MakeCommand("choose", chooseCommand),
		},
		nil,
		nil,
		nil,
		nil,
	)
}
