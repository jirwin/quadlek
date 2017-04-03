package lib


type Command interface {
	GetName() (string)
	RunCommand(bot *Bot, from string, to string, msg []string)
}

type Hook interface {
	RunHook(bot *Bot, from string, to string, msg []string)
}

type Plugin interface {
	GetCommands() ([]Command)
	RunCommands(bot *Bot, from string, to string, msg []string)
	RunHooks(bot *Bot, from string, to string, msg []string)
}
