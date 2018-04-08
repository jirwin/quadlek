package quadlek

import (
	"time"

	"fmt"

	"github.com/nlopes/slack"
)

type MessageLotOpts struct {
	IncludeBots     bool
	Count           int
	Period          time.Duration
	SkipAttachments bool
}

type LogMessage struct {
	Text string
}

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
