//go:generate protoc --go_out=. github.proto

package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	"github.com/jirwin/quadlek/quadlek"
	"golang.org/x/oauth2"
)

var scopes = []string{
	githu
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

func issueCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- nil

			title := cmdMsg.Command.Text
			if title == "" {
				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text: fmt.Sprintf("You must provide a title and description"),
				})
				continue
			}

			cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
				Text: fmt.Sprintf("Title: %s", title),
			})
		case <-ctx.Done():
			log.Info("Exiting github command")
			return
		}
	}
}

func githubAuthorizeWebhook(ctx context.Context, whChannel <-chan *quadlek.WebhookMsg) {
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
						Text: "Sorry! There was an error logging you into Github.",
					})
					return err
				}

				now := time.Now().UnixNano()
				if authState.ExpireTime < now {
					bkt.Delete([]byte("authstate-" + stateId[0]))
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Github.",
					})
					return errors.New("Received expired auth request")
				}

				auth := getSpotifyAuth()
				token, err := auth.Token(stateId[0], whMsg.Request)
				if err != nil {
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Github.",
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
		"github",
		[]quadlek.Command{quadlek.MakeCommand("issue", issueCommand)},
		nil,
		nil,
		[]quadlek.Webhook{quadlek.MakeWebhook("githubAuthorize", githubAuthorizeWebhook)},
		nil,
	)
}
