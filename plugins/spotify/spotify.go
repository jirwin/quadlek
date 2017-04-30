package spotify

import (
	"context"

	"github.com/jirwin/quadlek/quadlek"
	"log"
)

func spotifyAuthorizeWebhook(ctx context.Context, whChannel <-chan *quadlek.WebhookMsg) {
	for {
		select {
		case whMsg := <-whChannel:

			whMsg.Request.Body.Close()

		case <-ctx.Done():
			log.Info("Exiting spotify authorize command")
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"spotify",
		nil,
		nil,
		[]Webhook{quadlek.MakeWebhook("spotifyAuthorize")},
		load,
	)
}
