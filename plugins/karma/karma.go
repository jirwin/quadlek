package karma

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
	"github.com/nlopes/slack"
)

type KarmaScoreCommand struct{}

func (kh *KarmaScoreCommand) GetName() string {
	return "score"
}

func (kh *KarmaScoreCommand) RunCommand(bot *quadlek.Bot, msg *slack.Msg, parsedMsg string, store *quadlek.Store) {
	tokens := strings.Split(parsedMsg, " ")
	if len(tokens) != 1 {
		bot.Respond(msg, fmt.Sprintf("Invalid syntax. Example: %s score jirwin", bot.GetUserId()))
	}

	item := tokens[0]

	err := store.Get(item, func(val []byte) {
		var score string

		score = string(val)
		if val == nil {
			score = "0"
		}
		bot.Say(msg.Channel, fmt.Sprintf("%s: %s", item, score))
	})
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"item": item,
		}).Error("unable to get score")
		bot.Respond(msg, fmt.Sprintf("Unable to get score for %s.", item))
	}
}

type KarmaHook struct{}

var (
	ppRegex = regexp.MustCompile(".+\\+\\+$")
	mmRegex = regexp.MustCompile(".+--$")
)

func (kh *KarmaHook) RunHook(bot *quadlek.Bot, msg *slack.Msg, store *quadlek.Store) {
	tokens := strings.Split(msg.Text, " ")

	for _, t := range tokens {
		match := ppRegex.FindString(t)
		if match != "" {
			item := match[:len(match)-2]
			err := store.GetAndUpdate(item, func(val []byte) ([]byte, error) {
				if val == nil {
					return []byte("1"), nil
				}

				karma, err := strconv.Atoi(string(val[:]))
				if err != nil {
					return nil, err
				}

				bot.Respond(msg, fmt.Sprintf("Incremented %s", t))

				karma++
				karmaStr := strconv.Itoa(karma)

				return []byte(karmaStr), nil
			})
			if err != nil {
				log.WithFields(log.Fields{
					"err": err,
				}).Errorf("Error incrementing value: %s", t)
			}
		}

		match = mmRegex.FindString(t)
		if match != "" {
			item := match[:len(match)-2]
			err := store.GetAndUpdate(item, func(val []byte) ([]byte, error) {
				if val == nil {
					return []byte("-1"), nil
				}

				karma, err := strconv.Atoi(string(val[:]))
				if err != nil {
					return nil, err
				}

				karma--
				karmaStr := strconv.Itoa(karma)

				bot.Respond(msg, fmt.Sprintf("Decremented %s", t))

				return []byte(karmaStr), nil
			})
			if err != nil {
				log.WithFields(log.Fields{
					"err": err,
				}).Errorf("Error decrementing value: %s", t)
			}
		}
	}
}

type Plugin struct {
	Commands []quadlek.Command
	Hooks    []quadlek.Hook
}

func (p *Plugin) GetCommands() []quadlek.Command {
	return p.Commands
}

func (p *Plugin) GetHooks() []quadlek.Hook {
	return p.Hooks
}

func (p *Plugin) GetId() string {
	return "e0aee0d4-2b01-4549-a99b-02b0c8ba791f"
}

func Register() quadlek.Plugin {
	return &Plugin{
		Commands: []quadlek.Command{&KarmaScoreCommand{}},
		Hooks:    []quadlek.Hook{&KarmaHook{}},
	}
}
