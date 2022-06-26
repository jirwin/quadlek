package plugin_manager

import (
	"context"
	"net/http"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// Command is the interface that plugins implement for slash commands.
// Slash commands are actively triggered by users in slack, and only receive messages when they are invoked.
type Command interface {
	GetName() string
	Channel() chan<- *CommandMsg
	Run(ctx context.Context)
}

// registeredCommand is a struct used internally to represent a command that a plugin has registered
type registeredCommand struct {
	PluginId string
	Command  Command
}

// command is a an implementation of the Command interface
type command struct {
	name    string
	channel chan *CommandMsg
	runFunc func(ctx context.Context, cmdChan <-chan *CommandMsg)
}

// GetName returns the name of the command. This name should match the slash command configured in slack.
func (c *command) GetName() string {
	return c.name
}

// Channel returns the channel that the Bot will write incoming slash command messages to
func (c *command) Channel() chan<- *CommandMsg {
	return c.channel
}

// Run executes the commands runFunc with the provided context
func (c *command) Run(ctx context.Context) {
	c.runFunc(ctx, c.channel)
}

// MakeCommand is a helper function that accepts a name and a runFunc, and returns a Command.
func MakeCommand(name string, runFn func(ctx context.Context, cmdChan <-chan *CommandMsg)) Command {
	return &command{
		name:    name,
		runFunc: runFn,
		channel: make(chan *CommandMsg),
	}
}

// CommandMsg is the struct that is passed to a commands channel as it is activated.
type CommandMsg struct {
	Bot     *Bot
	Command *slashCommand
	Store   *Store
}

// CommandResp is the struct that is used to respond to a command if interaction is required.
type CommandResp struct {
	Text         string             `json:"text"`
	Attachments  []slack.Attachment `json:"attachments"`
	ResponseType string             `json:"response_type"`
	InChannel    bool               `json:"-"`
}

// Interaction is the interface that plugins implement for slash Shortcuts.
// Slash Shortcuts are actively triggered by users in slack, and only receive messages when they are invoked.
type Interaction interface {
	GetName() string
	Channel() chan<- *InteractionMsg
	Run(ctx context.Context)
}

// registeredInteraction is a struct used internally to represent a Interaction that a plugin has registered
type registeredInteraction struct {
	PluginId    string
	Interaction Interaction
}

// interaction is a an implementation of the Interaction interface
type interaction struct {
	name    string
	channel chan *InteractionMsg
	runFunc func(ctx context.Context, interactionChan <-chan *InteractionMsg)
}

// GetName returns the name of the Interaction. This name should match the slash Interaction configured in slack.
func (c *interaction) GetName() string {
	return c.name
}

// Channel returns the channel that the Bot will write incoming slash Interaction messages to
func (c *interaction) Channel() chan<- *InteractionMsg {
	return c.channel
}

// Run executes the Shortcuts runFunc with the provided context
func (c *interaction) Run(ctx context.Context) {
	c.runFunc(ctx, c.channel)
}

// MakeInteraction is a helper function that accepts a name and a runFunc, and returns a Interaction.
func MakeInteraction(name string, runFn func(ctx context.Context, cmdChan <-chan *InteractionMsg)) Interaction {
	return &interaction{
		name:    name,
		runFunc: runFn,
		channel: make(chan *InteractionMsg),
	}
}

// InteractionMsg is the struct that is passed to a Shortcuts channel as it is activated.
type InteractionMsg struct {
	Bot         *Bot
	Interaction *slack.InteractionCallback
	Store       *Store
}

// Hook is the interface that a plugin can implement to create a hook.
//
// Hooks receive every message that the Bot sees so plugins can react accordingly.
type Hook interface {
	Channel() chan<- *HookMsg
	Run(ctx context.Context)
}

// HookMsg is the struct that is passed to a hook's channel for each message seen.
type HookMsg struct {
	Bot   *Bot
	Msg   *slack.Msg
	Store *Store
}

// registeredHook is the struct used internally to represent a registered hook.
type registeredHook struct {
	PluginId string
	Hook     Hook
}

// hook is an internal implementation of the Hook interface.
type hook struct {
	channel chan *HookMsg
	runFunc func(ctx context.Context, hookChan <-chan *HookMsg)
}

// Channel returns the channel for the Bot to write HookMsg objects to.
func (h *hook) Channel() chan<- *HookMsg {
	return h.channel
}

// Run executes the hook's runFunc with the provided context.
func (h *hook) Run(ctx context.Context) {
	h.runFunc(ctx, h.channel)
}

// MakeHook is a helper function that accepts a runFunc and returns a Hook
func MakeHook(runFunc func(ctx context.Context, hookChan <-chan *HookMsg)) Hook {
	return &hook{
		channel: make(chan *HookMsg),
		runFunc: runFunc,
	}
}

// ReactionHook is the interface that plugins implement to create reaction hooks.
// Reaction hooks receive an event every time a message is reacted to.
type ReactionHook interface {
	Channel() chan<- *ReactionHookMsg
	Run(ctx context.Context)
}

// ReactionHookMsg is the struct that is sent to a reaction hook when a message is reacted to.
type ReactionHookMsg struct {
	Bot      *Bot
	Reaction *slackevents.ReactionAddedEvent
	Store    *Store
}

// registeredReactionHook is the internal struct that represents a registered plugin.
type registeredReactionHook struct {
	PluginId     string
	ReactionHook ReactionHook
}

// registeredHook is the internal struct that implements ReactionHook
type reactionHook struct {
	channel chan *ReactionHookMsg
	runFunc func(ctx context.Context, reactionHookChan <-chan *ReactionHookMsg)
}

// Channel returns the channel that the Bot writes ReactionHookMsgs to
func (r *reactionHook) Channel() chan<- *ReactionHookMsg {
	return r.channel
}

// Run executes the reaction hook's runFunc.
func (r *reactionHook) Run(ctx context.Context) {
	r.runFunc(ctx, r.channel)
}

// MakeReactionHook is a helper function that returns a ReactionHook
func MakeReactionHook(runFunc func(ctx context.Context, reactionHookChan <-chan *ReactionHookMsg)) ReactionHook {
	return &reactionHook{
		channel: make(chan *ReactionHookMsg),
		runFunc: runFunc,
	}
}

// Webhook is the interface that a plugin implements to register a custom webhook.
type Webhook interface {
	GetName() string
	Channel() chan<- *WebhookMsg
	Run(ctx context.Context)
}

// WebhookMsg is the struct that is sent to the plugin's channel
type WebhookMsg struct {
	Bot            *Bot
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Store          *Store
	Done           chan bool
}

// registeredWebhook is the internal struct that represents a registered webhook
type registeredWebhook struct {
	PluginId string
	Webhook  Webhook
}

// webhook is an implementation of the Webhook interface
type webhook struct {
	name    string
	channel chan *WebhookMsg
	runFunc func(ctx context.Context, webhookChan <-chan *WebhookMsg)
}

// GetName returns the name of the webhook
func (wh *webhook) GetName() string {
	return wh.name
}

// Channel returns the channel the Bot writes WebhookMsg when a custom webhook is received
func (wh *webhook) Channel() chan<- *WebhookMsg {
	return wh.channel
}

// Run executes the webhook's runFunc
func (wh *webhook) Run(ctx context.Context) {
	wh.runFunc(ctx, wh.channel)
}

// MakeWebhook is a helper function that returns a Webhook
func MakeWebhook(name string, runFunc func(ctx context.Context, whChan <-chan *WebhookMsg)) Webhook {
	return &webhook{
		name:    name,
		runFunc: runFunc,
		channel: make(chan *WebhookMsg),
	}
}

// Plugin is the interface to implement a plugin
type Plugin interface {
	GetId() string
}

type CommandPlugin interface {
	Plugin
	GetCommands() []Command
}

type HookPlugin interface {
	Plugin
	GetHooks() []Hook
}
type WebhookPlugin interface {
	Plugin
	GetWebhooks() []Webhook
}
type ReactionHookPlugin interface {
	Plugin
	GetReactionHooks() []ReactionHook
}
type LoadPlugin interface {
	Plugin
	Load(bot *Bot, store *Store) error
}

type InteractionPlugin interface {
	GetId() string
	GetInteractions() []Interaction
}
