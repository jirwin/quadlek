package karma

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
)

type KarmaScoreCommand struct {
	channel chan *quadlek.CommandMsg
}

func (kh *KarmaScoreCommand) GetName() string {
	return "score"
}

func (kh *KarmaScoreCommand) Channel() chan<- *quadlek.CommandMsg {
	return kh.channel
}

func (kh *KarmaScoreCommand) Run(ctx context.Context) {
	for {
		select {
		case cmdMsg := <-kh.channel:
			tokens := strings.Split(cmdMsg.ParsedMsg, " ")
			if len(tokens) != 1 {
				cmdMsg.Bot.Respond(cmdMsg.Msg, fmt.Sprintf("Invalid syntax. Example: %s score jirwin", cmdMsg.Bot.GetUserId()))
			}

			item := tokens[0]

			err := cmdMsg.Store.Get(item, func(val []byte) {
				var score string

				score = string(val)
				if val == nil {
					score = "0"
				}
				cmdMsg.Bot.Say(cmdMsg.Msg.Channel, fmt.Sprintf("%s: %s", item, score))
			})
			if err != nil {
				log.WithFields(log.Fields{
					"err":  err,
					"item": item,
				}).Error("unable to get score")
				cmdMsg.Bot.Respond(cmdMsg.Msg, fmt.Sprintf("Unable to get score for %s.", item))
			}

		case <-ctx.Done():
			log.Info("Exiting KarmaScoreCommand.")
			return
		}
	}
}

func MakeScoreCommand() quadlek.Command {
	return &KarmaScoreCommand{
		channel: make(chan *quadlek.CommandMsg),
	}
}

type KarmaHook struct {
	channel chan *quadlek.HookMsg
}

var (
	ppRegex = regexp.MustCompile(".+\\+\\+$")
	mmRegex = regexp.MustCompile(".+--$")
)

func (kh *KarmaHook) Channel() chan<- *quadlek.HookMsg {
	return kh.channel
}

func (kh *KarmaHook) Run(ctx context.Context) {
	for {
		select {
		case hookMsg := <-kh.channel:
			tokens := strings.Split(hookMsg.Msg.Text, " ")

			for _, t := range tokens {
				match := ppRegex.FindString(t)
				if match != "" {
					item := match[:len(match)-2]
					err := hookMsg.Store.GetAndUpdate(item, func(val []byte) ([]byte, error) {
						if val == nil {
							return []byte("1"), nil
						}

						karma, err := strconv.Atoi(string(val[:]))
						if err != nil {
							return nil, err
						}

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
					err := hookMsg.Store.GetAndUpdate(item, func(val []byte) ([]byte, error) {
						if val == nil {
							return []byte("-1"), nil
						}

						karma, err := strconv.Atoi(string(val[:]))
						if err != nil {
							return nil, err
						}

						karma--
						karmaStr := strconv.Itoa(karma)

						return []byte(karmaStr), nil
					})
					if err != nil {
						log.WithFields(log.Fields{
							"err": err,
						}).Errorf("Error decrementing value: %s", t)
					}
				}
			}

		case <-ctx.Done():
			log.Info("Exiting Karma Hook.")
			return
		}
	}
}

func MakeKarmaHook() quadlek.Hook {
	return &KarmaHook{
		channel: make(chan *quadlek.HookMsg),
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
		Commands: []quadlek.Command{MakeScoreCommand()},
		Hooks:    []quadlek.Hook{MakeKarmaHook()},
	}
}
