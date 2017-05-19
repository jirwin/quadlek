//go:generate protoc --go_out=. spotify.proto

package spotify

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"time"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	"github.com/jirwin/quadlek/quadlek"
	uuid "github.com/satori/go.uuid"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

var scopes = []string{
	spotify.ScopePlaylistModifyPublic,
	spotify.ScopeUserReadCurrentlyPlaying,
}

func (at *AuthToken) GetOauthToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  at.Token.AccessToken,
		TokenType:    at.Token.TokenType,
		RefreshToken: at.Token.RefreshToken,
		Expiry:       time.Unix(at.Token.ExpiresAt/9000000000, 0), //FIXME I accidentally stored this as nanos
	}
}

func (at *AuthToken) PopulateFromOauthToken(token *oauth2.Token) {
	at.Token = &Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry.UnixNano(),
	}
}

func startAuthFlow(stateId string) string {
	auth := getSpotifyAuth()
	url := auth.AuthURL(stateId)

	return url
}

func getSpotifyAuth() spotify.Authenticator {
	return spotify.NewAuthenticator(fmt.Sprintf("%s/%s", quadlek.WebhookRoot, "spotifyAuthorize"), scopes...)
}

func getSpotifyClient(authToken *AuthToken) (spotify.Client, bool) {

	auth := getSpotifyAuth()
	var token = authToken.GetOauthToken()
	client := auth.NewClient(token)

	if !reflect.DeepEqual(authToken.Scopes, scopes) {
		return client, true
	}

	_, err := client.CurrentUser()
	if err != nil {
		if strings.Contains(err.Error(), "token revoked") {
			return client, true
		}
	}
	return client, false
}

func authFlow(cmdMsg *quadlek.CommandMsg, bkt *bolt.Bucket) error {
	stateId := uuid.NewV4().String()
	authUrl := startAuthFlow(stateId)

	authState := &AuthState{
		Id:          stateId,
		UserId:      cmdMsg.Command.UserId,
		ResponseUrl: cmdMsg.Command.ResponseUrl,
		ExpireTime:  time.Now().UnixNano() + int64(time.Minute*15),
	}

	authStateBytes, err := proto.Marshal(authState)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error marshalling auth state")
		return err
	}

	err = bkt.Put([]byte("authstate-"+stateId), authStateBytes)
	if err != nil {
		cmdMsg.Command.Reply() <- &quadlek.CommandResp{
			Text: "There was an error authenticating to Spotify.",
		}
		return err
	}

	cmdMsg.Command.Reply() <- &quadlek.CommandResp{
		Text: fmt.Sprintf("You need to be authenticate to Spotify to continue. Please visit %s to do this.", authUrl),
	}
	return nil
}

func nowPlaying(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			err := cmdMsg.Store.UpdateRaw(func(bkt *bolt.Bucket) error {
				authToken := &AuthToken{}
				authTokenBytes := bkt.Get([]byte("authtoken-" + cmdMsg.Command.UserId))
				err := proto.Unmarshal(authTokenBytes, authToken)
				if err != nil {
					log.WithFields(log.Fields{
						"err": err,
					}).Error("error unmarshalling auth token")
					return err
				}

				if authToken.Token == nil {
					err = authFlow(cmdMsg, bkt)
					if err != nil {
						log.WithFields(log.Fields{
							"err": err,
						}).Error("error during auth flow")
						return err
					}
					return nil
				}

				client, needsReauth := getSpotifyClient(authToken)
				if needsReauth {
					err = authFlow(cmdMsg, bkt)
					if err != nil {
						return err
					}
					return nil
				}

				playing, err := client.PlayerCurrentlyPlaying()
				if err != nil {
					cmdMsg.Command.Reply() <- &quadlek.CommandResp{
						Text: "Unable to get currently playing.",
					}
					log.WithFields(log.Fields{
						"err": err,
					}).Error("error getting currently playing.")
					return err
				}

				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      fmt.Sprintf("<@%s> is listening to %s", cmdMsg.Command.UserId, playing.Item.URI),
					InChannel: true,
				}

				return nil
			})
			if err != nil {
				continue
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
			whMsg.Request.Body.Close()
			if !ok {
				log.WithFields(log.Fields{
					"url": whMsg.Request.URL.String(),
				}).Error("invalid callback url")
				continue
			}

			err := whMsg.Store.UpdateRaw(func(bkt *bolt.Bucket) error {
				authStateBytes := bkt.Get([]byte("authstate-" + stateId[0]))
				authState := &AuthState{}
				err := proto.Unmarshal(authStateBytes, authState)
				if err != nil {
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					return err
				}

				now := time.Now().UnixNano()
				if authState.ExpireTime < now {
					bkt.Delete([]byte("authstate-" + stateId[0]))
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					return errors.New("Received expired auth request")
				}

				auth := getSpotifyAuth()
				token, err := auth.Token(stateId[0], whMsg.Request)
				if err != nil {
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					return err
				}

				authToken := &AuthToken{}
				authToken.PopulateFromOauthToken(token)
				authToken.Scopes = scopes

				tokenBytes, err := proto.Marshal(authToken)
				err = bkt.Put([]byte("authtoken-"+authState.UserId), tokenBytes)
				if err != nil {
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					log.Error("error storing auth token.")
					return err
				}

				whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
					Text: "Successfully logged into Spotify. Try your command again please.",
				})

				return nil
			})
			if err != nil {
				log.WithFields(log.Fields{
					"err": err,
				}).Error("error authenticating to spotify")
				continue
			}

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
		[]quadlek.Hook{
			quadlek.MakeHook(saveSongsHook),
		},
		nil,
		[]quadlek.Webhook{
			quadlek.MakeWebhook("spotifyAuthorize", spotifyAuthorizeWebhook),
		},
		nil,
	)
}
