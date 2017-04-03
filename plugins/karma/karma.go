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

type KarmaHook struct{}

var (
	ppRegex = regexp.MustCompile(".+\\+\\+$")
	mmRegex = regexp.MustCompile(".+--$")
)

func (kh *KarmaHook) RunHook(bot *quadlek.Bot, msg *slack.Msg, store *quadlek.Store) {
	tokens := strings.Split(msg.Text, " ")

	for _, t := range tokens {
		go func(token string) {
			if match := ppRegex.FindString(t); match != "" {
				item := match[:len(match)-2]
				err := store.GetAndUpdate(item, func(val []byte) ([]byte, error) {
					if val == nil {
						return []byte("1"), nil
					}

					karma, err := strconv.Atoi(string(val[:]))
					if err != nil {
						return nil, err
					}

					bot.Respond(msg, fmt.Sprintf("Incremented %s", item))

					karma++
					karmaStr := strconv.Itoa(karma)

					return []byte(karmaStr), nil
				})
				if err != nil {
					log.WithFields(log.Fields{
						"err": err,
					}).Errorf("Error incrementing value: %s", item)
				}
			} else if match := mmRegex.FindString(t); match != "" {
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

					bot.Respond(msg, fmt.Sprintf("Decremented %s", item))

					return []byte(karmaStr), nil
				})
				if err != nil {
					log.WithFields(log.Fields{
						"err": err,
					}).Errorf("Error decrementing value: %s", item)
				}
			}
		}(t)
	}
}

type Plugin struct {
	Hooks []quadlek.Hook
}

func (p *Plugin) GetCommands() []quadlek.Command {
	return []quadlek.Command{}
}

func (p *Plugin) GetHooks() []quadlek.Hook {
	return p.Hooks
}

func (p *Plugin) GetId() string {
	return "e0aee0d4-2b01-4549-a99b-02b0c8ba791f"
}

func Register() quadlek.Plugin {
	return &Plugin{
		Hooks: []quadlek.Hook{&KarmaHook{}},
	}
}
