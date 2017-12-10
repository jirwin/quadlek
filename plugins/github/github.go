//go:generate protoc --go_out=. github.proto

package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	"reflect"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-github/github"
	"github.com/jirwin/quadlek/quadlek"
	"github.com/satori/go.uuid"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

var (
	scopes       = []string{"user:email", "repo"}
	clientId     string
	clientSecret string
	defaultOwner string
)

func (at *AuthToken) GetOauthToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  at.Token.AccessToken,
		TokenType:    at.Token.TokenType,
		RefreshToken: at.Token.RefreshToken,
		Expiry:       time.Unix(at.Token.ExpiresAt, 0),
	}
}

func (at *AuthToken) PopulateFromOauthToken(token *oauth2.Token) {
	at.Token = &Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry.Unix(),
	}
}

func getGithubOauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       scopes,
		Endpoint:     githuboauth.Endpoint,
	}
}

func startAuthFlow(stateId string) string {
	conf := getGithubOauthConfig()
	return conf.AuthCodeURL(stateId)
}

func authFlow(cmdMsg *quadlek.CommandMsg, bkt *bolt.Bucket) error {
	stateId := uuid.NewV4().String()
	authUrl := startAuthFlow(stateId)

	authState := &AuthState{
		Id:          stateId,
		UserId:      cmdMsg.Command.UserId,
		ResponseUrl: cmdMsg.Command.ResponseUrl,
		ExpireTime:  time.Now().Unix() + int64(time.Minute*15),
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
			Text: "There was an error authenticating to Github",
		}
		return err
	}

	cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
		Text: fmt.Sprintf("You need to authenticate to Github to continue. Please visit %s to do this.", authUrl),
	})

	return nil
}

func getGithubClient(ctx context.Context, authToken *AuthToken) (*github.Client, bool) {
	token := authToken.GetOauthToken()

	if !reflect.DeepEqual(authToken.Scopes, scopes) {
		return nil, true
	}
	oauthClient := getGithubOauthConfig().Client(ctx, token)
	client := github.NewClient(oauthClient)

	_, _, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
		}).Error("User doesn't seem to be authenticated.")
		return nil, true
	}

	return client, false
}

func issueCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			cmdMsg.Command.Reply() <- nil

			msg := strings.SplitN(cmdMsg.Command.Text, " ", 2)
			if len(msg) != 2 {
				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text: fmt.Sprintf("You must provide a repo and issue title. ex: /issue jirwin/quadlek Make me better!"),
				})
				continue
			}

			var owner string
			var repo string
			repoParts := strings.Split(msg[0], "/")
			if len(repoParts) == 2 {
				owner = repoParts[0]
				repo = repoParts[1]
			} else if len(repoParts) == 1 {
				if defaultOwner == "" {
					cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
						Text: fmt.Sprintf("You didn't specify an org for the repo, and no default org was defined."),
					})
				}
				owner = defaultOwner
				repo = repoParts[0]
			}

			title := msg[1]
			if title == "" {
				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text: fmt.Sprintf("You must provide a title."),
				})
				continue
			}

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

				if authToken.GithubUser == "" {
					err = authFlow(cmdMsg, bkt)
					if err != nil {
						log.WithError(err).Error("error during authflow")
						return err
					}
					return nil
				}

				client, needsReauth := getGithubClient(ctx, authToken)
				if needsReauth {
					err = authFlow(cmdMsg, bkt)
					if err != nil {
						return err
					}
					return nil
				}

				body := fmt.Sprintf("%s created this issue from slack", authToken.GithubUser)
				issue, _, err := client.Issues.Create(ctx, owner, repo, &github.IssueRequest{
					Title: &title,
					Body:  &body,
				})
				if err != nil {
					log.WithError(err).Error("Error creating issue.")
					return err
				}

				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text:      fmt.Sprintf("%s created a new issue: %s", authToken.GithubUser, issue.GetHTMLURL()),
					InChannel: true,
				})

				return nil
			})

			if err != nil {
				continue
			}

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
			state := whMsg.Request.FormValue("state")
			whMsg.Request.Body.Close()

			err := whMsg.Store.UpdateRaw(func(bkt *bolt.Bucket) error {
				authStateBytes := bkt.Get([]byte("authstate-" + state))
				authState := &AuthState{}
				err := proto.Unmarshal(authStateBytes, authState)
				if err != nil {
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Github.",
					})
					return err
				}

				now := time.Now().Unix()
				if authState.ExpireTime < now {
					bkt.Delete([]byte("authstate-" + state))
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Github.",
					})
					return errors.New("received expired auth request")
				}

				if state != authState.Id {
					log.Errorf("invalid oauth state, expected '%s', got '%s'\n", authState.Id, state)
					return errors.New("received invalid oauth state")
				}

				oauthConfig := getGithubOauthConfig()

				code := whMsg.Request.FormValue("code")
				token, err := oauthConfig.Exchange(ctx, code)
				if err != nil {
					log.WithError(err).Error("oauth exchange failed")
					return err
				}

				oauthClient := oauthConfig.Client(ctx, token)
				client := github.NewClient(oauthClient)
				user, _, err := client.Users.Get(ctx, "")
				if err != nil {
					log.WithError(err).Error("failed to get user")
					return err
				}

				authToken := &AuthToken{}
				authToken.PopulateFromOauthToken(token)
				authToken.Scopes = scopes
				authToken.GithubUser = user.GetLogin()

				tokenBytes, err := proto.Marshal(authToken)
				err = bkt.Put([]byte("authtoken-"+authState.UserId), tokenBytes)
				if err != nil {
					whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
						Text: "Sorry! There was an error logging you into Github.",
					})
					log.Error("error storing auth token.")
					return err
				}

				whMsg.Bot.RespondToSlashCommand(authState.ResponseUrl, &quadlek.CommandResp{
					Text: fmt.Sprintf("Successfully logged into Github as %s. Try your command again please.", authToken.GithubUser),
				})

				return nil
			})
			if err != nil {
				log.WithFields(log.Fields{
					"err": err,
				}).Error("error authenticating to github")
				continue
			}

		case <-ctx.Done():
			log.Info("Exiting github authorize command")
			return
		}
	}
}

func Register(id, secret, owner string) quadlek.Plugin {
	clientId = id
	clientSecret = secret
	defaultOwner = owner

	return quadlek.MakePlugin(
		"github",
		[]quadlek.Command{quadlek.MakeCommand("issue", issueCommand)},
		nil,
		nil,
		[]quadlek.Webhook{quadlek.MakeWebhook("githubAuthorize", githubAuthorizeWebhook)},
		nil,
	)
}
