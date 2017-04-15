package quadlek

import (
	"errors"
	"fmt"
	"strings"

	"context"

	"bytes"
	"encoding/json"
	"net/http"

	"time"

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

type Hook interface {
	Channel() chan<- *HookMsg
	Run(ctx context.Context)
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

type HookMsg struct {
	Bot   *Bot
	Msg   *slack.Msg
	Store *Store
}

type registeredHook struct {
	PluginId string
	Hook     Hook
}

type Plugin interface {
	GetId() string
	GetCommands() []Command
	GetHooks() []Hook
	Load(bot *Bot, store *Store) error
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
		go func() {
			b.wg.Add(1)
			defer b.wg.Done()

			command.Run(b.ctx)
		}()
	}

	for _, hook := range plugin.GetHooks() {
		b.hooks = append(b.hooks, &registeredHook{
			PluginId: plugin.GetId(),
			Hook:     hook,
		})
		go func() {
			b.wg.Add(1)
			defer b.wg.Done()

			hook.Run(b.ctx)
		}()
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

func (b *Bot) dispatchHooks(msg *slack.Msg) {
	for _, hook := range b.hooks {
		hook.Hook.Channel() <- &HookMsg{
			Bot:   b,
			Msg:   msg,
			Store: b.getStore(hook.PluginId),
		}
	}
}

func (b *Bot) handleSlackCommand(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error parsing form. Invalid slack command hook.")
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	cmd := &slashCommand{}
	err = decoder.Decode(cmd, r.PostForm)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error marshalling slack command.")
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	if cmd.Token != b.verificationToken {
		log.Error("Invalid validation token was used. Ignoring.")
		generateErrorMsg(w, "Sorry. I was unable to complete your request. :cry:")
		return
	}

	respChan := make(chan *CommandResp)
	cmd.ResponseChan = respChan
	b.cmdChannel <- cmd

	timer := time.NewTimer(time.Millisecond * 2900)
	for {
		select {
		case resp := <-respChan:
			if timer.Stop() {
				prepareSlashCommandResp(resp)
				jsonResponse(w, resp)
			} else {
				b.RespondToSlashCommand(cmd.ResponseUrl, resp)
			}
			return

		case <-timer.C:
			log.Info("Didn't get a response soon enough. Moving on.")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{})
			return
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
