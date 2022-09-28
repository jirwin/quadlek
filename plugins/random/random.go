package random

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
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

// diceRegex matches the 1d6 format for dice rolling
var diceRegex = regexp.MustCompile("([0-9]+)[dD]([0-9]+)(?:[+]([0-9]+))?")

func diceCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			text := cmdMsg.Command.Text
			choices := strings.Split(text, " ")

			if len(choices) == 0 {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text: "I can't roll zero dice!",
				}
				continue
			}

			// Find all matches for the dice regex
			found := diceRegex.FindAllStringSubmatch(text, -1)
			if len(found) == 0 {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text: "I don't understand your fancy dice. Try sending me things like `1d6 2d4` or `11d12+2`.",
				}
				continue
			}

			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text:      extractAndRollDice(found),
				InChannel: true,
			}

		case <-ctx.Done():
			return
		}
	}
}

func extractAndRollDice(matches [][]string) string {
	rv := "You rolled:"

	for _, match := range matches {
		count, _ := strconv.Atoi(match[1])
		sides, _ := strconv.Atoi(match[2])

		add := 0
		addTxt := ""
		// if we have that match
		if len(match) == 4 {
			// Parse it, add to the string
			add, _ = strconv.Atoi(match[3])
			if add != 0 {
				addTxt = "+" + match[3]
			}
		}

		var vals []string
		total := int64(0)
		for i := 0; i < count; i++ {
			// Actually roll it
			val := rand.Int63n(int64(sides)) + 1 + int64(add)
			total += val
			vals = append(vals, fmt.Sprintf("%d", val))
		}

		rv += fmt.Sprintf("\n%dd%d: %s %s = %d", count, sides, total, addTxt, total+int64(add))
	}

	return rv
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"random",
		[]quadlek.Command{
			quadlek.MakeCommand("roll", rollCommand),
			quadlek.MakeCommand("choose", chooseCommand),
			quadlek.MakeCommand("dice", diceCommand),
		},
		nil,
		nil,
		nil,
		nil,
	)
}
