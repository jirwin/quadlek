package twitter

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/jirwin/quadlek/quadlek"
)

func load(consumerKey, consumerSecret, accessToken, accessSecret string, filter map[string]string) func(bot *quadlek.Bot, store *quadlek.Store) error {

	return func(bot *quadlek.Bot, store *quadlek.Store) error {
		go func() {
			config := oauth1.NewConfig(consumerKey, consumerSecret)
			token := oauth1.NewToken(accessToken, accessSecret)
			httpClient := config.Client(oauth1.NoContext, token)
			client := twitter.NewClient(httpClient)

			followFilters := []string{}
			for follow := range filter {
				followFilters = append(followFilters, follow)
			}

			filterParams := &twitter.StreamFilterParams{
				Follow:        followFilters,
				StallWarnings: twitter.Bool(true),
			}

			stream, err := client.Streams.Filter(filterParams)
			if err != nil {
				zap.L().Error("Error streaming tweets.", zap.Error(err))
				return
			}

			for msg := range stream.Messages {
				switch m := msg.(type) {
				case *twitter.Tweet:
					if channel, ok := filter[m.User.IDStr]; ok {
						if m.RetweetedStatus != nil {
							zap.L().Info("Got a tweet containing a retweet", zap.Any("tweet", m))
							if replyChannel, ok := filter[m.RetweetedStatus.User.IDStr]; ok && channel == replyChannel {
								zap.L().Info("Tweet contains retweet from already monitored account, cancelling message", zap.Any("tweet", m))
								continue
							}
						}
						twitterUrl := fmt.Sprintf("https://twitter.com/%s/status/%s", m.User.ScreenName, m.IDStr)
						chanId, err := bot.GetChannelId(channel)
						if err != nil {
							zap.L().Error("unable to find channel.", zap.Error(err))
							continue
						}
						bot.Say(chanId, twitterUrl)
					}
				}
			}
		}()

		return nil
	}
}

func Register(consumerKey, consumerSecret, accessToken, accessSecret string, filter map[string]string) quadlek.Plugin {
	return quadlek.MakePlugin(
		"quadlek-twitter",
		nil,
		nil,
		nil,
		nil,
		load(consumerKey, consumerSecret, accessToken, accessSecret, filter),
	)
}
