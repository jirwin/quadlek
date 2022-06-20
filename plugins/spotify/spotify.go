//go:generate protoc --go_out=. spotify.proto

package spotify

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/jirwin/quadlek/pb/quadlek/plugins/spotify/v1"
	"net/http"
	"os"
	"reflect"

	"go.uber.org/zap"

	"time"

	"strings"

	"github.com/boltdb/bolt"
	"github.com/jirwin/quadlek/quadlek"
	uuid "github.com/satori/go.uuid"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

const WebhookRoot = "https://%s/slack/plugin"

var scopes = []string{
	spotify.ScopePlaylistModifyPublic,
	spotify.ScopeUserReadCurrentlyPlaying,
}

func GetOauthToken(at *v1.AuthToken) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  at.Token.AccessToken,
		TokenType:    at.Token.TokenType,
		RefreshToken: at.Token.RefreshToken,
		Expiry:       time.Unix(at.Token.ExpiresAt/9000000000, 0), //FIXME I accidentally stored this as nanos
	}
}

func PopulateFromOauthToken(at *v1.AuthToken, token *oauth2.Token) {
	at.Token = &v1.Token{
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

func webhookRoot() string {
	return fmt.Sprintf(WebhookRoot, os.Getenv("SPOTIFY_WEBHOOK_DOMAIN"))
}

func getSpotifyAuth() spotify.Authenticator {
	return spotify.NewAuthenticator(fmt.Sprintf("%s/%s", webhookRoot(), "spotifyAuthorize"), scopes...)
}

func getSpotifyClient(authToken *v1.AuthToken) (spotify.Client, bool) {

	auth := getSpotifyAuth()
	var token = GetOauthToken(authToken)
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
	uuid := uuid.NewV4()

	stateId := uuid.String()
	authUrl := startAuthFlow(stateId)

	authState := &v1.AuthState{
		Id:          stateId,
		UserId:      cmdMsg.Command.UserId,
		ResponseUrl: cmdMsg.Command.ResponseUrl,
		ExpireTime:  time.Now().UnixNano() + int64(time.Minute*15),
	}

	authStateBytes, err := proto.Marshal(authState)
	if err != nil {
		zap.L().Error("error marshalling auth state", zap.Error(err))
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
				authToken := &v1.AuthToken{}
				authTokenBytes := bkt.Get([]byte("authtoken-" + cmdMsg.Command.UserId))
				err := proto.Unmarshal(authTokenBytes, authToken)
				if err != nil {
					zap.L().Error("error unmarshalling auth token", zap.Error(err))
					return err
				}

				if authToken.Token == nil {
					err = authFlow(cmdMsg, bkt)
					if err != nil {
						zap.L().Error("error during auth flow", zap.Error(err))
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
					zap.L().Error("error getting currently playing.", zap.Error(err))
					return err
				}

				if playing != nil && playing.Item != nil {
					cmdMsg.Command.Reply() <- &quadlek.CommandResp{
						Text:      fmt.Sprintf("<@%s> is listening to %s", cmdMsg.Command.UserId, playing.Item.URI),
						InChannel: true,
					}
				}

				return nil
			})
			if err != nil {
				continue
			}

		case <-ctx.Done():
			zap.L().Info("Exiting NowPlayingCommand.")
			return
		}
	}
}

func spotifyAuthorizeWebhook(ctx context.Context, whChannel <-chan *quadlek.WebhookMsg) {
	for {
		select {
		case whMsg := <-whChannel:
			// respond to webhook
			whMsg.ResponseWriter.WriteHeader(http.StatusOK)
			_, err := whMsg.ResponseWriter.Write([]byte{})
			if err != nil {
				continue
			}
			whMsg.Done <- true

			// process webhook
			query := whMsg.Request.URL.Query()
			stateId, ok := query["state"]
			whMsg.Request.Body.Close()
			if !ok {
				zap.L().Error("invalid callback url")
				continue
			}

			err = whMsg.Store.UpdateRaw(func(bkt *bolt.Bucket) error {
				authStateBytes := bkt.Get([]byte("authstate-" + stateId[0]))
				authState := &v1.AuthState{}
				err := proto.Unmarshal(authStateBytes, authState)
				if err != nil {
					_ = whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					return err
				}

				now := time.Now().UnixNano()
				if authState.ExpireTime < now {
					err = bkt.Delete([]byte("authstate-" + stateId[0]))
					if err != nil {
						return err
					}
					_ = whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					return errors.New("Received expired auth request")
				}

				auth := getSpotifyAuth()
				token, err := auth.Token(stateId[0], whMsg.Request)
				if err != nil {
					_ = whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					return err
				}

				authToken := &v1.AuthToken{}
				PopulateFromOauthToken(authToken, token)
				authToken.Scopes = scopes

				tokenBytes, err := proto.Marshal(authToken)
				if err != nil {
					_ = whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					return err
				}
				err = bkt.Put([]byte("authtoken-"+authState.UserId), tokenBytes)
				if err != nil {
					_ = whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Spotify.",
					})
					zap.L().Error("error storing auth token.")
					return err
				}

				_ = whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
					Text: "Successfully logged into Spotify. Try your command again please.",
				})

				return nil
			})
			if err != nil {
				zap.L().Error("error authenticating to spotify", zap.Error(err))
				continue
			}

		case <-ctx.Done():
			zap.L().Info("Exiting spotify authorize command")
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
