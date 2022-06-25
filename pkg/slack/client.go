package slack

import "github.com/slack-go/slack"

type SlackClient struct {
	client *slack.Client
}

func (s *SlackClient) Client() *slack.Client {
	return s.client
}

func NewSlackClient(config Config, httpClient *slackHttpClient) (*SlackClient, error) {
	opts := []slack.Option{
		slack.OptionHTTPClient(httpClient),
	}
	if config.Debug {
		opts = append(opts, slack.OptionDebug(true))
	}

	c := &SlackClient{
		client: slack.New(config.ApiKey, opts...),
	}

	return c, nil
}
