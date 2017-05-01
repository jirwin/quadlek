package spotify

import (
	"context"
	"fmt"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
	"github.com/satori/go.uuid"
	"github.com/zmb3/spotify"
)

func startAuthFlow() string {
	auth := spotify.NewAuthenticator(fmt.Sprintf("%s", path.Join(quadlek.WebhookRoot, "spotifyAuthorize")), spotify.ScopePlaylistModifyPublic, spotify.ScopePlaylistModifyPrivate, spotify.ScopeUserReadCurrentlyPlaying)

	url := auth.AuthURL(uuid.NewV4().String())

	return url
}

func nowPlaying(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			err := cmdMsg.Store.Get("authorization-"+cmdMsg.Command.UserId, func(val []byte) error {
				authToken := string(val)
				if authToken == "" {
					authUrl := startAuthFlow()

					cmdMsg.Command.Reply() <- &quadlek.CommandResp{
						Text: fmt.Sprintf("You need to be authenticate to Spotify to continue. Please visit %s to do this.", authUrl),
					}
					return nil
				}

				return nil
			})
			if err != nil {
				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text: "Unable to run now playing.",
				})
			}

		case <-ctx.Done():
			log.Info("Exiting NowPlayingCommand.")
			return
		}
	}
}

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
		[]quadlek.Command{
			quadlek.MakeCommand("nowplaying", nowPlaying),
		},
		nil,
		[]quadlek.Webhook{
			quadlek.MakeWebhook("spotifyAuthorize", spotifyAuthorizeWebhook),
		},
		nil,
	)
}
