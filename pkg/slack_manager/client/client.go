package client

import (
	"fmt"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
	"os"
)

type Config struct {
	ApiKey            string
	Debug             bool
	RequestTracing    bool
	VerificationToken string
}

func NewConfig() (Config, error) {
	c := Config{}
	apiKey := os.Getenv("SLACK_API_KEY")
	if apiKey == "" {
		return Config{}, fmt.Errorf("SLACK_API_KEY must be set")
	}
	c.ApiKey = apiKey

	verificationToken := os.Getenv("SLACK_VERIFICATION_TOKEN")
	if verificationToken == "" {
		return Config{}, fmt.Errorf("SLACK_VERIFICATION_TOKEN must be set")
	}
	c.VerificationToken = verificationToken

	debug := os.Getenv("SLACK_DEBUG")
	if debug != "" {
		c.Debug = true
	}

	reqTracing := os.Getenv("SLACK_REQUEST_TRACING")
	if reqTracing != "" {
		c.RequestTracing = true
	}

	return c, nil
}

type SlackClient interface {
	Api() *slack.Client
	Respond(msg *slack.Msg, resp string)
	PostMessage(channel, resp string, params slack.PostMessageParameters) (string, string, error)
	OpenModalView(triggerID string, response slack.ModalViewRequest) (*slack.ViewResponse, error)
	Say(channel string, resp string)
	React(msg *slack.Msg, reaction string)
}

type SlackClientImpl struct {
	api *slack.Client
	l   *zap.Logger
}

func (s *SlackClientImpl) Api() *slack.Client {
	return s.api
}

// Respond responds to a SlackManager message
// The sent message will go to the same channel as the message that is being responded to and will highlight
// the author of the original message.
func (s *SlackClientImpl) Respond(msg *slack.Msg, resp string) {
	s.api.PostMessage(msg.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s>: %s", msg.User, resp), false)) //nolint:errcheck
}

// PostMessage sends a new message to SlackManager using the provided channel and message string.
// It returns the channel ID the message was posted to, and the timestamp that the message was posted at.
// In combination these can be used to identify the exact message that was sent.
func (s *SlackClientImpl) PostMessage(channel, resp string, params slack.PostMessageParameters) (string, string, error) {
	return s.api.PostMessage(channel, slack.MsgOptionText(resp, false))
}

// Say sends a message to the provided channel
func (s *SlackClientImpl) Say(channel string, resp string) {
	s.api.PostMessage(channel, slack.MsgOptionText(resp, false)) //nolint:errcheck
}

// OpenView uses a trigger_id to open the provided view Request
func (s *SlackClientImpl) OpenModalView(triggerID string, response slack.ModalViewRequest) (*slack.ViewResponse, error) {
	r, err := s.api.OpenView(triggerID, response)
	if err != nil {
		s.l.Error("error opening view", zap.Error(err))
		return nil, err
	}

	return r, nil
}

// React attaches an emojii reaction to a message.
// Reactions are formatted like: :+1:
func (s *SlackClientImpl) React(msg *slack.Msg, reaction string) {
	s.api.AddReaction(reaction, slack.NewRefToMessage(msg.Channel, msg.Timestamp)) //nolint:errcheck
}

func NewSlackClient(config Config, l *zap.Logger, httpClient *slackHttpClient) (*SlackClientImpl, error) {
	opts := []slack.Option{
		slack.OptionHTTPClient(httpClient),
	}
	if config.Debug {
		opts = append(opts, slack.OptionDebug(true))
	}

	c := &SlackClientImpl{
		api: slack.New(config.ApiKey, opts...),
		l:   l,
	}

	return c, nil
}
