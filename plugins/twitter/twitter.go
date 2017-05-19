package twitter

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
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
			for follow, _ := range filter {
				followFilters = append(followFilters, follow)
			}

			filterParams := &twitter.StreamFilterParams{
				Follow:        followFilters,
				StallWarnings: twitter.Bool(true),
			}

			stream, err := client.Streams.Filter(filterParams)
			if err != nil {
				log.WithField("err", err).Error("Error streaming tweets.")
				return
			}

			for {
				select {
				case msg := <-stream.Messages:
					switch m := msg.(type) {
					case *twitter.Tweet:
						if channel, ok := filter[m.User.IDStr]; ok {
							twitterUrl := fmt.Sprintf("https://twitter.com/%s/status/%s", m.User.ScreenName, m.IDStr)
							chanId, err := bot.GetChannelId(channel)
							if err != nil {
								log.WithField("err", err).Error("unable to find channel.")
								continue
							}
							bot.Say(chanId, twitterUrl)
						}
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
