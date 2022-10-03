package infobot

import (
	"context"
	"fmt"
	"strings"

	"github.com/jirwin/quadlek/quadlek"

	"go.uber.org/zap"
)

var factStore = MakeFactStore()

const FactStoreKey = "facts"

func load(bot *quadlek.Bot, store *quadlek.Store) error {
	return store.Get(FactStoreKey, func(rec []byte) error {
		return factStore.Load(rec)
	})
}

func infobot(ctx context.Context, hookChan <-chan *quadlek.HookMsg) {

	for {
		select {

		case hookMsg := <-hookChan:
			if !hookMsg.Bot.MsgToBot(hookMsg.Msg.Text) {
				continue
			}

			line := strings.TrimPrefix(hookMsg.Msg.Text, fmt.Sprintf("<@%s> ", hookMsg.Bot.GetUserId()))

			if lookup := factStore.LookupFact(line); lookup != "" {
				hookMsg.Bot.Respond(hookMsg.Msg, lookup)
				continue
			}

			if factStore.HumanFactSet(line) {
				out, err := factStore.Serialize()
				if err != nil {
					zap.L().Error("error serializing factstore", zap.Error(err))
					continue
				}

				err = hookMsg.Store.Update(FactStoreKey, out)

				if err != nil {
					zap.L().Error("error while saving factstore", zap.Error(err))
					continue
				}

				hookMsg.Bot.Respond(hookMsg.Msg, "Alright. "+line)
				continue
			}

			if factStore.HumanFactForget(line) {
				out, err := factStore.Serialize()
				if err != nil {
					zap.L().Error("error serializing factstore", zap.Error(err))
					continue
				}

				err = hookMsg.Store.Update(FactStoreKey, out)

				if err != nil {
					zap.L().Error("error while saving factstore", zap.Error(err))
					continue
				}

				hookMsg.Bot.Respond(hookMsg.Msg, "Alright. I forgot it.")
				continue
			}

		case <-ctx.Done():
			zap.L().Info("Shutting down infobot hook.")
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"infobot",
		nil,
		[]quadlek.Hook{
			quadlek.MakeHook(infobot),
		},
		nil,
		nil,
		load,
	)
}
