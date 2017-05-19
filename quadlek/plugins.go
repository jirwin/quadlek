package quadlek

import (
	"errors"
	"fmt"
	"strings"

	"context"

	"bytes"
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
)

type Command interface {
	GetName() string
	Channel() chan<- *CommandMsg
	Run(ctx context.Context)
}

type registeredCommand struct {
	PluginId string
	Command  Command
}

type command struct {
	name    string
	channel chan *CommandMsg
	runFunc func(ctx context.Context, cmdChan <-chan *CommandMsg)
}

func (c *command) GetName() string {
	return c.name
}

func (c *command) Channel() chan<- *CommandMsg {
	return c.channel
}

func (c *command) Run(ctx context.Context) {
	c.runFunc(ctx, c.channel)
}

func MakeCommand(name string, runFn func(ctx context.Context, cmdChan <-chan *CommandMsg)) Command {
	return &command{
		name:    name,
		runFunc: runFn,
		channel: make(chan *CommandMsg),
	}
}

type CommandMsg struct {
	Bot     *Bot
	Command *slashCommand
	Store   *Store
}

type CommandResp struct {
	Text         string             `json:"text"`
	Attachments  []slack.Attachment `json:"attachments"`
	ResponseType string             `json:"response_type"`
	InChannel    bool               `json:"-"`
}

type Hook interface {
	Channel() chan<- *HookMsg
	Run(ctx context.Context)
}

type HookMsg struct {
	Bot   *Bot
	Msg   *slack.Msg
	Store *Store
}

type registeredHook struct {
	PluginId string
	Hook     Hook
}

type hook struct {
	channel chan *HookMsg
	runFunc func(ctx context.Context, hookChan <-chan *HookMsg)
}

func (h *hook) Channel() chan<- *HookMsg {
	return h.channel
}

func (h *hook) Run(ctx context.Context) {
	h.runFunc(ctx, h.channel)
}

func MakeHook(runFunc func(ctx context.Context, hookChan <-chan *HookMsg)) Hook {
	return &hook{
		channel: make(chan *HookMsg),
		runFunc: runFunc,
	}
}

type ReactionHook interface {
	Channel() chan<- *ReactionHookMsg
	Run(ctx context.Context)
}

type ReactionHookMsg struct {
	Bot      *Bot
	Reaction *slack.ReactionAddedEvent
	Store    *Store
}

type registeredReactionHook struct {
	PluginId     string
	ReactionHook ReactionHook
}

type reactionHook struct {
	channel chan *ReactionHookMsg
	runFunc func(ctx context.Context, reactionHookChan <-chan *ReactionHookMsg)
}

func (r *reactionHook) Channel() chan<- *ReactionHookMsg {
	return r.channel
}

func (r *reactionHook) Run(ctx context.Context) {
	r.runFunc(ctx, r.channel)
}

func MakeReactionHook(runFunc func(ctx context.Context, reactionHookChan <-chan *ReactionHookMsg)) ReactionHook {
	return &reactionHook{
		channel: make(chan *ReactionHookMsg),
		runFunc: runFunc,
	}
}

type Webhook interface {
	GetName() string
	Channel() chan<- *WebhookMsg
	Run(ctx context.Context)
}

type WebhookMsg struct {
	Bot     *Bot
	Request *http.Request
	Store   *Store
}

type registeredWebhook struct {
	PluginId string
	Webhook  Webhook
}

type webhook struct {
	name    string
	channel chan *WebhookMsg
	runFunc func(ctx context.Context, webhookChan <-chan *WebhookMsg)
}

func (wh *webhook) GetName() string {
	return wh.name
}

func (wh *webhook) Channel() chan<- *WebhookMsg {
	return wh.channel
}

func (wh *webhook) Run(ctx context.Context) {
	wh.runFunc(ctx, wh.channel)
}

func MakeWebhook(name string, runFunc func(ctx context.Context, whChan <-chan *WebhookMsg)) Webhook {
	return &webhook{
		name:    name,
		runFunc: runFunc,
		channel: make(chan *WebhookMsg),
	}
}

type Plugin interface {
	GetId() string
	GetCommands() []Command
	GetHooks() []Hook
	GetWebhooks() []Webhook
	GetReactionHooks() []ReactionHook
	Load(bot *Bot, store *Store) error
}

type loadPluginFn func(bot *Bot, store *Store) error

type plugin struct {
	id            string
	commands      []Command
	hooks         []Hook
	reactionHooks []ReactionHook
	webhooks      []Webhook
	loadFn        loadPluginFn
}

func (p *plugin) GetId() string {
	return p.id
}

func (p *plugin) GetCommands() []Command {
	return p.commands
}

func (p *plugin) GetHooks() []Hook {
	return p.hooks
}

func (p *plugin) GetWebhooks() []Webhook {
	return p.webhooks
}

func (p *plugin) GetReactionHooks() []ReactionHook {
	return p.reactionHooks
}

func (p *plugin) Load(bot *Bot, store *Store) error {
	return p.loadFn(bot, store)
}

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

func (b *Bot) MsgToBot(msg string) bool {
	return strings.HasPrefix(msg, fmt.Sprintf("<@%s> ", b.userId))
}

func (b *Bot) GetCommand(cmdText string) *registeredCommand {
	if cmdText == "" {
		return nil
	}

	if cmd, ok := b.commands[cmdText]; ok {
		return cmd
	}

	return nil
}

func (b *Bot) GetWebhook(name string) *registeredWebhook {
	if name == "" {
		return nil
	}

	if wh, ok := b.webhooks[name]; ok {
		return wh
	}

	return nil
}

func (b *Bot) getStore(pluginId string) *Store {
	return &Store{
		db:       b.db,
		pluginId: pluginId,
	}
}

func (b *Bot) RegisterPlugin(plugin Plugin) error {
	if plugin.GetId() == "" {
		return errors.New("Must provide a unique plugin id.")
	}

	err := b.InitPluginBucket(plugin.GetId())
	if err != nil {
		return err
	}

	err = plugin.Load(b, b.getStore(plugin.GetId()))

	for _, command := range plugin.GetCommands() {
		_, ok := b.commands[command.GetName()]
		if ok {
			return errors.New(fmt.Sprintf("Command already exists: %s", command.GetName()))
		}
		b.commands[command.GetName()] = &registeredCommand{
			PluginId: plugin.GetId(),
			Command:  command,
		}
		go func(c Command) {
			b.wg.Add(1)
			defer b.wg.Done()

			c.Run(b.ctx)
		}(command)
	}

	for _, hook := range plugin.GetHooks() {
		b.hooks = append(b.hooks, &registeredHook{
			PluginId: plugin.GetId(),
			Hook:     hook,
		})
		go func(h Hook) {
			b.wg.Add(1)
			defer b.wg.Done()

			h.Run(b.ctx)
		}(hook)
	}

	for _, reactionHook := range plugin.GetReactionHooks() {
		b.reactionHooks = append(b.reactionHooks, &registeredReactionHook{
			PluginId:     plugin.GetId(),
			ReactionHook: reactionHook,
		})
		go func(r ReactionHook) {
			b.wg.Add(1)
			defer b.wg.Done()

			r.Run(b.ctx)
		}(reactionHook)
	}

	for _, webhook := range plugin.GetWebhooks() {
		_, ok := b.webhooks[webhook.GetName()]
		if ok {
			return errors.New(fmt.Sprintf("Webhook already exists: %s", webhook.GetName()))
		}
		b.webhooks[webhook.GetName()] = &registeredWebhook{
			PluginId: plugin.GetId(),
			Webhook:  webhook,
		}
		go func(wh Webhook) {
			b.wg.Add(1)
			defer b.wg.Done()

			wh.Run(b.ctx)
		}(webhook)
	}

	return nil
}

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

func (b *Bot) dispatchWebhook(webhook *PluginWebhook) {
	wh := b.GetWebhook(webhook.Name)
	if wh == nil {
		return
	}

	wh.Webhook.Channel() <- &WebhookMsg{
		Bot:     b,
		Request: webhook.Request,
		Store:   b.getStore(wh.PluginId),
	}
}

func (b *Bot) dispatchReactions(ev *slack.ReactionAddedEvent) {
	for _, reactionHook := range b.reactionHooks {
		reactionHook.ReactionHook.Channel() <- &ReactionHookMsg{
			Bot:      b,
			Reaction: ev,
			Store:    b.getStore(reactionHook.PluginId),
		}
	}
}

func (b *Bot) dispatchHooks(msg *slack.Msg) {
	for _, hook := range b.hooks {
		hook.Hook.Channel() <- &HookMsg{
			Bot:   b,
			Msg:   msg,
			Store: b.getStore(hook.PluginId),
		}
	}
}

func prepareSlashCommandResp(cmd *CommandResp) {
	if cmd.ResponseType == "" {
		if cmd.InChannel {
			cmd.ResponseType = "in_channel"
		} else {
			cmd.ResponseType = "ephemeral"
		}
	}
}

func (b *Bot) RespondToSlashCommand(url string, cmdResp *CommandResp) error {
	prepareSlashCommandResp(cmdResp)

	jsonBytes, err := json.Marshal(cmdResp)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error marshalling json.")
		return err
	}
	data := bytes.NewBuffer(jsonBytes)
	json.NewEncoder(data).Encode(cmdResp)
	resp, err := http.Post(url, "application/json", data)
	defer resp.Body.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error responding to slash command.")
		return err
	}
	return nil
}
