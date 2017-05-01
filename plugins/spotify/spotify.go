//go:generate protoc --go_out=. spotify.proto

package spotify

import (
	"context"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
	uuid "github.com/satori/go.uuid"
	"github.com/zmb3/spotify"
	"time"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
)

func startAuthFlow(stateId string) string {
	auth := spotify.NewAuthenticator(fmt.Sprintf("%s/%s", quadlek.WebhookRoot, "spotifyAuthorize"), spotify.ScopePlaylistModifyPublic, spotify.ScopePlaylistModifyPrivate, spotify.ScopeUserReadCurrentlyPlaying)

	url := auth.AuthURL(stateId)

	return url
}

func nowPlaying(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			err := cmdMsg.Store.GetAndUpdate("authorization-"+cmdMsg.Command.UserId, func(val []byte) ([]byte, error) {
				authState := &AuthState{}
				err := proto.Unmarshal(val, authState)
				if err != nil {
					log.WithFields(log.Fields{
						"err": err,
					}).Error("error unmarshalling auth state")
					return nil, err
				}

				if authState.Token == "" {
					stateId := uuid.NewV4().String()
					authUrl := startAuthFlow(stateId)

					authState := &AuthState{
						Id: stateId,
						UserId: cmdMsg.Command.UserId,
						ResponseUrl: cmdMsg.Command.ResponseUrl,
						ExpireTime: time.Now().UnixNano() + int64(time.Minute * 15),
					}

					authStateBytes, err := proto.Marshal(authState)
					if err != nil {
						log.WithFields(log.Fields{
							"err": err,
						}).Error("error marshalling auth state")
						return nil, err
					}

					cmdMsg.Command.Reply() <- &quadlek.CommandResp{
						Text: fmt.Sprintf("You need to be authenticate to Spotify to continue. Please visit %s to do this.", authUrl),
					}
					return authStateBytes, nil
				}

				return nil, nil
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
			query := whMsg.Request.URL.Query()
			stateId, ok := query["state"]
			if !ok {
				log.WithFields(log.Fields{
					"url": whMsg.Request.URL.String(),
				}).Error("invalid callback url")
				continue
			}

			whMsg.Store.Update("authorization-"+)
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
