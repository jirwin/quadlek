package plugin_manager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"go.uber.org/zap"
)

func (m *ManagerImpl) handleSlackEvent(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.l.Error("unable to read event body")
		return
	}

	ev, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		m.l.Error("unable to parse event", zap.String("event", string(body)))
		return
	}

	switch ev.Type {
	case slackevents.URLVerification:
		urlEvent, ok := ev.Data.(*slackevents.EventsAPIURLVerificationEvent)
		if !ok {
			m.l.Error("unexpected data type for url validation")
			return
		}
		headers := w.Header()
		headers.Set("Content-type", "text/plain")
		_, _ = w.Write([]byte(urlEvent.Challenge))

	case slackevents.CallbackEvent:
		switch iev := ev.InnerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			// Ignore things the bot has said
			if iev.BotID != m.slackManager.GetBotId() {
				hookMsg := &slack.Msg{
					Channel:         iev.Channel,
					User:            iev.User,
					Text:            iev.Text,
					Timestamp:       iev.TimeStamp,
					ThreadTimestamp: iev.ThreadTimeStamp,
					BotID:           iev.BotID,
					Username:        iev.Username,
				}
				m.dispatchHooks(hookMsg)
			}

		case *slackevents.AppMentionEvent:
			if iev.User != "" {
				// Hack to handle slash commands from messages
				if strings.HasPrefix(iev.Text, fmt.Sprintf("<@%s> ", m.slackManager.GetUserId())) {
					tokens := strings.Split(iev.Text, " ")
					if len(tokens) > 1 {
						cmdName := tokens[1]
						cmd := m.getCommand(cmdName)
						if cmd == nil {
							return
						}
						channel, err := m.slackManager.GetChannel(iev.Channel)
						if err != nil {
							m.l.Error("error getting channel", zap.Error(err), zap.String("channel", iev.Channel))
							return
						}
						user, err := m.slackManager.GetUser(iev.User)
						if err != nil {
							m.l.Error("error getting user", zap.Error(err))
							return
						}
						slashCmd := &slashCommand{
							TeamId:      ev.TeamID,
							ChannelId:   channel.ID,
							ChannelName: channel.Name,
							UserId:      user.ID,
							UserName:    user.Name,
							Command:     tokens[1],
							Text:        strings.Join(tokens[2:], " "),
						}

						respChan := make(chan *CommandResp)
						slashCmd.responseChan = respChan

						go func() {
							timer := time.NewTimer(time.Millisecond * 2500)

							for {
								select {
								case <-timer.C:
									return

								case resp := <-respChan:
									timer.Stop()
									if resp != nil {
										msgOpts := []slack.MsgOption{
											slack.MsgOptionText(resp.Text, false),
											slack.MsgOptionAttachments(resp.Attachments...),
										}

										if !resp.InChannel {
											msgOpts = append(msgOpts, slack.MsgOptionPostEphemeral(iev.User))
										}

										_, _, _ = m.slackManager.Slack().Api().PostMessage(iev.Channel, msgOpts...)
										return
									}

									return
								}
							}
						}()

						cmd.Command.Channel() <- &CommandMsg{
							Bot:     b,
							Command: slashCmd,
							Store:   b.getStore(cmd.PluginId),
						}

					}
				}
			}

		case *slackevents.ReactionAddedEvent:
			m.dispatchReactions(iev)

		case *slackevents.MemberJoinedChannelEvent:
			if iev.User == m.slackManager.GetUserId() {
				m.slackManager.Slack().Say(iev.Channel, fmt.Sprintf("Thanks for inviting me <@%s>. I'm alive!", iev.Inviter))
			}

		case *slack.ChannelCreatedEvent:
			if iev.Channel.IsChannel {
				channel, err := m.slackManager.Slack().Api().GetConversationInfo(iev.Channel.ID, false)
				if err != nil {
					m.l.Error("Unable to add channel", zap.Error(err))
					return
				}
				m.slackManager.UpdateChannel(iev.Channel.ID, *channel)
			}

		case *slack.UserChangeEvent:
			m.slackManager.UpdateUser(iev.User.ID, iev.User)

		default:
			m.l.Debug("unhandled event", zap.Any("event", iev))
		}
	}
}
