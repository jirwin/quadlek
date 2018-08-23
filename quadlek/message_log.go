package quadlek

import (
	"time"

	"fmt"

	"github.com/nlopes/slack"
)

// MessageLotOpts is the stuct that you use to configure what messages you want to retrieve from the API.
//
// IncludeBots: If true, include messages from bots(not just quadlek bots)
//
// Count: The max number of messages to return
//
// Period: The amount of time to look backwards when looking for messages
//
// SkipAttachments: If true, don't return message attachments.
type MessageLotOpts struct {
	IncludeBots     bool
	Count           int
	Period          time.Duration
	SkipAttachments bool
}

// GetMessageLog uses channel and a set of options to get historical messages from the Slack API.
func (b *Bot) GetMessageLog(channel string, opts MessageLotOpts) ([]slack.Message, error) {
	params := slack.NewHistoryParameters()
	if opts.Count != 0 {
		params.Count = opts.Count
	}
	if opts.Period != time.Duration(0) {
		oldest := time.Now().UTC().Add(opts.Period*-1).UnixNano() / 1000000
		oldestTs := fmt.Sprintf("%d.000", oldest)
		params.Oldest = oldestTs
	}

	history, err := b.api.GetChannelHistory(channel, params)
	if err != nil {
		return nil, err
	}

	msgs := []slack.Message{}
	for _, msg := range history.Messages {
		if !opts.IncludeBots && msg.SubType == "bot_message" {
			continue
		}

		if opts.SkipAttachments && len(msg.Attachments) != 0 {
			continue
		}

		if msg.SubType != "" {
			continue
		}

		msgs = append(msgs, msg)
	}

	return msgs, nil
}
