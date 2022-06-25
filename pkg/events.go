package quadlek

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

func (b *Bot) handleSlackEvent(w http.ResponseWriter, r *http.Request) {
	err := b.ValidateSlackRequest(r)
	if err != nil {
		b.Log.Error("unable to validate request signature")
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.Log.Error("unable to read event body")
		return
	}

	ev, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		b.Log.Error("unable to parse event", zap.String("event", string(body)))
		return
	}

	switch ev.Type {
	case slackevents.URLVerification:
		urlEvent, ok := ev.Data.(*slackevents.EventsAPIURLVerificationEvent)
		if !ok {
			b.Log.Error("unexpected data type for url validation")
			return
		}
		headers := w.Header()
		headers.Set("Content-type", "text/plain")
		_, _ = w.Write([]byte(urlEvent.Challenge))

	case slackevents.CallbackEvent:
		switch iev := ev.InnerEvent.Data.(type) {

		case *slackevents.MessageEvent:
			// Ignore things the bot has said
			if iev.BotID != b.GetBotId() {
				hookMsg := &slack.Msg{
					Channel:         iev.Channel,
					User:            iev.User,
					Text:            iev.Text,
					Timestamp:       iev.TimeStamp,
					ThreadTimestamp: iev.ThreadTimeStamp,
					BotID:           iev.BotID,
					Username:        iev.Username,
				}
				b.dispatchHooks(hookMsg)
			}

		case *slackevents.AppMentionEvent:
			if iev.User != "" {
				// Hack to handle slash commands from messages
				if strings.HasPrefix(iev.Text, fmt.Sprintf("<@%s> ", b.userId)) {
					tokens := strings.Split(iev.Text, " ")
					if len(tokens) > 1 {
						cmdName := tokens[1]
						cmd := b.GetCommand(cmdName)
						if cmd == nil {
							return
						}
						channel, err := b.GetChannel(iev.Channel)
						if err != nil {
							b.Log.Error("error getting channel", zap.Error(err), zap.String("channel", iev.Channel))
							return
						}
						user, err := b.GetUser(iev.User)
						if err != nil {
							b.Log.Error("error getting user", zap.Error(err))
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

										_, _, _ = b.api.PostMessage(iev.Channel, msgOpts...)
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
			b.dispatchReactions(iev)

		case *slackevents.MemberJoinedChannelEvent:
			if iev.User == b.userId {
				b.Say(iev.Channel, fmt.Sprintf("Thanks for inviting me <@%s>. I'm alive!", iev.Inviter))
			}

		case *slack.ChannelCreatedEvent:
			if iev.Channel.IsChannel {
				channel, err := b.api.GetConversationInfo(iev.Channel.ID, false)
				if err != nil {
					b.Log.Error("Unable to add channel", zap.Error(err))
					return
				}
				b.humanChannels[channel.Name] = *channel
			}

		case *slack.UserChangeEvent:
			b.users[iev.User.ID] = iev.User
			b.humanUsers[iev.User.Name] = iev.User

		default:
			b.Log.Info("unhandled event", zap.Any("event", iev))
		}
	}
}
