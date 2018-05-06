package twitch

import (
	"context"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
)

var clientId = "d9ngrbqgxo96af92kvjp6xaluzv3dx"

//curl -H 'Client-ID: uo6dggojyb8d6soh92zknwmi5ej1q2' \
//-H 'Content-Type: application/json' \
//-X POST -d '{"hub.mode":"subscribe",
//"hub.topic":"https://api.twitch.tv/helix/users/follows?to_id=1337",
//"hub.callback":"https://yourwebsite.com/path/to/callback/handler",
//"hub.lease_seconds":"864000",
//"hub.secret": s3cRe7}' \
//https://api.twitch.tv/helix/webhooks/hub
type TwitchSubscribeReq struct {
	HubMode string `json:"hub.mode"`
	HubTopic string `json:"hub.topic"`
	HubCallback string `json:"hub.callback"`
	HubLeaseSeconds string `json:"hub.lease_seconds"`
	HubSecret string `json:"hub.secret"`
}

type TwitchUserInfo struct {
	Id      string
	Name    string
	Channel string
}

func twitchWebhook(users map[string]*TwitchUserInfo) func(context.Context, <-chan *quadlek.WebhookMsg) {
	return func(ctx context.Context, whChannel <-chan *quadlek.WebhookMsg) {

		for _, userInfo := range users {

		}

		renewSubscriptionTicker := time.NewTicker(time.Hour * 12)

		for {
			select {
			case wh := <-whChannel:
				wh.Request.
			case <-ctx.Done():
				log.Info("exiting twitch webbhook")
				return
			}
		}
	}
}

func Register(users map[string]*TwitchUserInfo) quadlek.Plugin {
	return quadlek.MakePlugin(
		"quadlek-twitch",
		nil,
		nil,
		nil,
		[]quadlek.Webhook{
			quadlek.MakeWebhook("twitch-webhook", twitchWebhook(users)),
		},
		nil,
	)
}
