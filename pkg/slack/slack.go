package slack

import (
	"fmt"
	"os"

	"github.com/slack-go/slack"
)

type SlackAPI interface {
	Client() *slack.Client
}

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
