package quadlek

import "github.com/nlopes/slack"

type Command interface {
	GetName() string
	RunCommand(bot *Bot, msg *slack.Msg, parsedMsg string)
}

type Hook interface {
	RunHook(bot *Bot, msg *slack.Msg)
}

type Plugin interface {
	GetCommands() []Command
	GetHooks() []Hook
}
