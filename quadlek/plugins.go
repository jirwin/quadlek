package quadlek

import (
	"errors"
	"fmt"
	"github.com/slack-go/slack/slackevents"
	"strings"

	"go.uber.org/zap"

	"context"

	"bytes"
	"encoding/json"
	"net/http"

	"github.com/slack-go/slack"
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
	GetCommands() []Command
	GetHooks() []Hook
	GetWebhooks() []Webhook
	GetReactionHooks() []ReactionHook
	Load(bot *Bot, store *Store) error
}

// loadPluginFn is used to do any initialization work when the plugin is loaded
type loadPluginFn func(bot *Bot, store *Store) error

// plugin is an internal implementation of Plugin
type plugin struct {
	id            string
	commands      []Command
	hooks         []Hook
	reactionHooks []ReactionHook
	webhooks      []Webhook
	loadFn        loadPluginFn
}

// GetId returns the id set by the plugin. This should be unique across plugins.
func (p *plugin) GetId() string {
	return p.id
}

// GetCommands returns all of the commands registered with the plugin.
func (p *plugin) GetCommands() []Command {
	return p.commands
}

// GetHooks returns all of the hooks registered with the plugin.
func (p *plugin) GetHooks() []Hook {
	return p.hooks
}

// GetWebhooks returns all of the webhooks registered with the plugin
func (p *plugin) GetWebhooks() []Webhook {
	return p.webhooks
}

// GetReactionHooks returns all of the reaction hooks registered with the plugin.
func (p *plugin) GetReactionHooks() []ReactionHook {
	return p.reactionHooks
}

// Load executes the load function specified by the plugin
func (p *plugin) Load(bot *Bot, store *Store) error {
	return p.loadFn(bot, store)
}

// MakePlugin is a helper function that returns a Plugin.
func MakePlugin(id string, commands []Command, hooks []Hook, reactionHooks []ReactionHook, webhooks []Webhook, loadFunction loadPluginFn) Plugin {
	if loadFunction == nil {
		loadFunction = func(bot *Bot, store *Store) error {
			return nil
		}
	}

	return &plugin{
		id:            id,
		commands:      commands,
		hooks:         hooks,
		webhooks:      webhooks,
		reactionHooks: reactionHooks,
		loadFn:        loadFunction,
	}
}

// MsgToBot returns true if the message was intended for the Bot
func (b *Bot) MsgToBot(msg string) bool {
	return strings.HasPrefix(msg, fmt.Sprintf("<@%s> ", b.userId))
}

// GetCommand returns the registeredCommand for the provided command name
func (b *Bot) GetCommand(cmdText string) *registeredCommand {
	if cmdText == "" {
		return nil
	}

	if cmd, ok := b.commands[cmdText]; ok {
		return cmd
	}

	return nil
}

// GetWebhook returns the registeredWebhook for the given webhook name
func (b *Bot) GetWebhook(name string) *registeredWebhook {
	if name == "" {
		return nil
	}

	if wh, ok := b.webhooks[name]; ok {
		return wh
	}

	return nil
}

// getStore returns the database handle for the given pluginId
func (b *Bot) getStore(pluginId string) *Store {
	return &Store{
		db:       b.db,
		pluginId: pluginId,
	}
}

// RegisterPlugin registers the given Plugin with the Bot.
func (b *Bot) RegisterPlugin(plugin Plugin) error {
	if plugin.GetId() == "" {
		return errors.New("Must provide a unique plugin id.")
	}

	err := b.InitPluginBucket(plugin.GetId())
	if err != nil {
		return err
	}

	err = plugin.Load(b, b.getStore(plugin.GetId()))
	if err != nil {
		return err
	}

	for _, command := range plugin.GetCommands() {
		_, ok := b.commands[command.GetName()]
		if ok {
			return fmt.Errorf("Command already exists: %s", command.GetName())
		}
		b.commands[command.GetName()] = &registeredCommand{
			PluginId: plugin.GetId(),
			Command:  command,
		}
		b.wg.Add(1)
		go func(c Command) {
			defer b.wg.Done()

			c.Run(b.ctx)
		}(command)
	}

	for _, hook := range plugin.GetHooks() {
		b.hooks = append(b.hooks, &registeredHook{
			PluginId: plugin.GetId(),
			Hook:     hook,
		})
		b.wg.Add(1)
		go func(h Hook) {
			defer b.wg.Done()

			h.Run(b.ctx)
		}(hook)
	}

	for _, reactionHook := range plugin.GetReactionHooks() {
		b.reactionHooks = append(b.reactionHooks, &registeredReactionHook{
			PluginId:     plugin.GetId(),
			ReactionHook: reactionHook,
		})
		b.wg.Add(1)
		go func(r ReactionHook) {
			defer b.wg.Done()

			r.Run(b.ctx)
		}(reactionHook)
	}

	for _, webhook := range plugin.GetWebhooks() {
		_, ok := b.webhooks[webhook.GetName()]
		if ok {
			return fmt.Errorf("Webhook already exists: %s", webhook.GetName())
		}
		b.webhooks[webhook.GetName()] = &registeredWebhook{
			PluginId: plugin.GetId(),
			Webhook:  webhook,
		}
		b.wg.Add(1)
		go func(wh Webhook) {
			defer b.wg.Done()

			wh.Run(b.ctx)
		}(webhook)
	}

	return nil
}

// dispatchCommand parses an incoming slash command and sends it to the plugin it is registered to
func (b *Bot) dispatchCommand(slashCmd *slashCommand) {
	if slashCmd.Command == "" {
		return
	}
	cmdName := slashCmd.Command[1:]

	cmd := b.GetCommand(cmdName)
	if cmd == nil {
		return
	}

	cmd.Command.Channel() <- &CommandMsg{
		Bot:     b,
		Command: slashCmd,
		Store:   b.getStore(cmd.PluginId),
	}
}

// dispatchWebhook parses an incoming webhook and sends it to the plugin it is registered to
func (b *Bot) dispatchWebhook(webhook *PluginWebhook) {
	wh := b.GetWebhook(webhook.Name)
	if wh == nil {
		return
	}

	wh.Webhook.Channel() <- &WebhookMsg{
		Bot:            b,
		Request:        webhook.Request,
		ResponseWriter: webhook.ResponseWriter,
		Store:          b.getStore(wh.PluginId),
	}
}

// dispatchReactions sends a reaction to all registered reaction hooks
func (b *Bot) dispatchReactions(ev *slackevents.ReactionAddedEvent) {
	for _, reactionHook := range b.reactionHooks {
		reactionHook.ReactionHook.Channel() <- &ReactionHookMsg{
			Bot:      b,
			Reaction: ev,
			Store:    b.getStore(reactionHook.PluginId),
		}
	}
}

// dispatchHooks sends a slack message to all registered hooks
func (b *Bot) dispatchHooks(msg *slack.Msg) {
	for _, hook := range b.hooks {
		hook.Hook.Channel() <- &HookMsg{
			Bot:   b,
			Msg:   msg,
			Store: b.getStore(hook.PluginId),
		}
	}
}

// prepareSlashCommandResp prepares a command response for API submission
func prepareSlashCommandResp(cmd *CommandResp) {
	if cmd.ResponseType == "" {
		if cmd.InChannel {
			cmd.ResponseType = "in_channel"
		} else {
			cmd.ResponseType = "ephemeral"
		}
	}
}

// RespondToSlashCommand sends a command response to the slack API in order to respond to a slash command.
func (b *Bot) RespondToSlashCommand(url string, cmdResp *CommandResp) error {
	prepareSlashCommandResp(cmdResp)

	jsonBytes, err := json.Marshal(cmdResp)
	if err != nil {
		b.Log.Error("error marshalling json.", zap.Error(err))
		return err
	}
	data := bytes.NewBuffer(jsonBytes)
	err = json.NewEncoder(data).Encode(cmdResp)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err != nil {
		b.Log.Error("error responding to slash command.", zap.Error(err))
		return err
	}
	return nil
}
