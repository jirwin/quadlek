package slack_manager

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/slack_manager/client"
)

type Config struct {
}

func NewConfig() (Config, error) {
	c := Config{}
	return c, nil
}

type slackState struct {
	sync.RWMutex

	UserID        string
	BotID         string
	Channels      map[string]slack.Channel
	HumanChannels map[string]slack.Channel
	Users         map[string]slack.User
	HumanUsers    map[string]slack.User
}

func newSlackState(userID, botID string) *slackState {
	return &slackState{
		UserID:        userID,
		BotID:         botID,
		Channels:      make(map[string]slack.Channel),
		HumanChannels: make(map[string]slack.Channel),
		Users:         make(map[string]slack.User),
		HumanUsers:    make(map[string]slack.User),
	}
}

type Manager interface {
	Start(ctx context.Context) error
	Done() <-chan struct{}
	Stop()
	UpdateUser(user slack.User)
	UpdateChannel(channel slack.Channel)
	GetUserId() string
	GetBotId() string
	GetChannelId(chanName string) (string, error)
	GetChannel(chanID string) (slack.Channel, error)
	GetUser(userID string) (slack.User, error)
	GetUserName(userID string) (string, error)
	Slack() client.SlackClient
}

type ManagerImpl struct {
	l          *zap.Logger
	c          Config
	slack      client.SlackClient
	slackState *slackState
	ctx        context.Context
	cancel     context.CancelFunc
}

func (m *ManagerImpl) Done() <-chan struct{} {
	return m.ctx.Done()
}

func (m *ManagerImpl) Stop() {
	m.cancel()
}

func (m *ManagerImpl) UpdateUser(user slack.User) {
	m.slackState.Lock()
	defer m.slackState.Unlock()

	m.slackState.Users[user.ID] = user
	m.slackState.HumanUsers[user.Name] = user
}

func (m *ManagerImpl) UpdateChannel(channel slack.Channel) {
	m.slackState.Lock()
	defer m.slackState.Unlock()

	m.slackState.Channels[channel.ID] = channel
	m.slackState.HumanChannels[channel.Name] = channel
}

func (m *ManagerImpl) Slack() client.SlackClient {
	return m.slack
}

func (m *ManagerImpl) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	at, err := m.slack.Api().AuthTestContext(m.ctx)
	if err != nil {
		m.l.Error("Unable to auth", zap.Error(err))
		return err
	}

	m.slackState = newSlackState(at.UserID, at.BotID)
	m.slackState.Lock()
	defer m.slackState.Unlock()

	pageToken := ""
	for {
		channels, nextPage, err := m.slack.Api().GetConversationsContext(m.ctx, &slack.GetConversationsParameters{Cursor: pageToken})
		if err != nil {
			m.l.Error("Unable to list channels", zap.Error(err))
			return err
		}
		for _, channel := range channels {
			m.UpdateChannel(channel)
		}

		if nextPage == "" {
			break
		}
		pageToken = nextPage
	}

	users, err := m.slack.Api().GetUsersContext(m.ctx)
	if err != nil {
		m.l.Error("Unable to list users", zap.Error(err))
		return err
	}
	for _, user := range users {
		m.UpdateUser(user)
	}

	if v := os.Getenv("COMMIT_SHA"); v != "" {
		if c, ok := m.slackState.HumanChannels["qdev"]; ok {
			m.slack.Say(c.ID, fmt.Sprintf("I'm back. My version is %s", v))
		}
	}

	return nil
}

// GetUserId returns the SlackManager user ID for the Bot.
func (m *ManagerImpl) GetUserId() string {
	m.slackState.RLock()
	defer m.slackState.RUnlock()

	return m.slackState.UserID
}

// GetBotId returns the SlackManager bot ID
func (m *ManagerImpl) GetBotId() string {
	m.slackState.RLock()
	defer m.slackState.RUnlock()

	return m.slackState.BotID
}

// GetChannelId returns the SlackManager channel ID for a given human-readable channel name.
func (m *ManagerImpl) GetChannelId(chanName string) (string, error) {
	m.slackState.RLock()
	defer m.slackState.RUnlock()

	channel, ok := m.slackState.HumanChannels[chanName]
	if !ok {
		return "", fmt.Errorf("Channel(%s) not found.", chanName)
	}

	return channel.ID, nil
}

// GetChannel returns the SlackManager channel object given a channel ID
func (m *ManagerImpl) GetChannel(chanID string) (slack.Channel, error) {
	m.slackState.RLock()
	defer m.slackState.RUnlock()

	channel, ok := m.slackState.Channels[chanID]
	if !ok {
		return slack.Channel{}, fmt.Errorf("Channel(%s) not found.", chanID)
	}

	return channel, nil
}

// GetUser returns the SlackManager user object given a user ID
func (m *ManagerImpl) GetUser(userID string) (slack.User, error) {
	m.slackState.RLock()
	defer m.slackState.RUnlock()

	user, ok := m.slackState.Users[userID]
	if !ok {
		return slack.User{}, fmt.Errorf("User(%s) not found.", userID)
	}

	return user, nil
}

// GetUserName returns the human-readable user name for a given user ID
func (m *ManagerImpl) GetUserName(userID string) (string, error) {
	m.slackState.RLock()
	defer m.slackState.RUnlock()

	user, ok := m.slackState.Users[userID]
	if !ok {
		return "", fmt.Errorf("User(%s) not found.", userID)
	}

	return user.Name, nil
}

func New(l *zap.Logger, c Config, slackClient client.SlackClient) (*ManagerImpl, error) {
	m := &ManagerImpl{
		l:     l.Named("slack-manager"),
		c:     c,
		slack: slackClient,
	}

	return m, nil
}
