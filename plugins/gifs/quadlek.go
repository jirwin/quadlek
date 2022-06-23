package gifs

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/segmentio/fasthash/fnv1a"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	v1 "github.com/jirwin/quadlek/pb/quadlek/plugins/gifs/v1"
	"github.com/jirwin/quadlek/quadlek"
)

var gifs *Gifs

const (
	GoodBotReaction = "good-bot"
	BadBotReaction  = "bad-bot"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func newReply(phrase string, url string) *v1.Reply {
	return &v1.Reply{
		Phrase:    phrase,
		Timestamp: time.Now().UnixNano(),
		Url:       url,
	}
}

func saveAlias(bkt *bolt.Bucket, alias *v1.Alias) error {
	out, err := proto.Marshal(alias)
	if err != nil {
		return err
	}

	err = bkt.Put([]byte(getAliasName(alias.Phrase)), out)
	if err != nil {
		return err
	}

	return nil
}

func getAlias(bkt *bolt.Bucket, phrase string) (*v1.Alias, bool, error) {
	out := bkt.Get(getAliasName(phrase))

	if out == nil {
		return nil, false, nil
	}

	a, err := parseAlias(out)
	if err != nil {
		return nil, false, err
	}

	return a, true, nil
}

func parseAlias(b []byte) (*v1.Alias, error) {
	a := &v1.Alias{}
	err := proto.Unmarshal(b, a)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func diffAppendReply(newReply *v1.Reply, replies []*v1.Reply) []*v1.Reply {
	rMap := make(map[string]*v1.Reply, len(replies))
	rMap[newReply.Url] = newReply
	for _, r := range replies {
		rMap[r.Url] = r
	}

	rv := make([]*v1.Reply, 0, len(replies))
	for _, r := range rMap {
		rv = append(rv, r)
	}

	return rv
}

//func removeReply(reply *v1.Reply, replies []*v1.Reply) []*v1.Reply {
//	rv := make([]*v1.Reply, 0, len(replies))
//	for _, r := range replies {
//		if r.Url == reply.Url {
//			continue
//		}
//		rv = append(rv, r)
//	}
//
//	return rv
//}

func replyMatch(url string, replies []*v1.Reply) bool {
	for _, r := range replies {
		if url == r.Url {
			return true
		}
	}

	return false
}

func getAliasName(phrase string) []byte {
	return []byte(fmt.Sprintf("alias:%d", fnv1a.HashString64(phrase)))
}

func appendUrl(bkt *bolt.Bucket, phrase string, url string, block bool, forceNew bool) error {
	var alias *v1.Alias
	var ok bool
	var err error

	if !forceNew {
		alias, ok, err = getAlias(bkt, phrase)
		if err != nil {
			return err
		}
	}

	reply := newReply(phrase, url)

	if ok {
		if block {
			alias.Blocked = diffAppendReply(reply, alias.Blocked)
		} else {
			alias.Allowed = diffAppendReply(reply, alias.Allowed)
		}
	} else {
		alias = &v1.Alias{
			Phrase: phrase,
		}
		if block {
			alias.Blocked = diffAppendReply(reply, alias.Blocked)
		} else {
			alias.Allowed = diffAppendReply(reply, alias.Allowed)
		}
	}
	err = saveAlias(bkt, alias)
	if err != nil {
		return err
	}

	return nil
}

func gifLoad(bot *quadlek.Bot, store *quadlek.Store) error {
	migrated := false
	err := store.Get("meta:v2Migration", func(b []byte) error {
		if b != nil {
			migrated = true
			return nil
		}

		return nil
	})
	if migrated {
		return nil
	}

	err = store.ForEach(func(bkt *bolt.Bucket, key string, value []byte) error {
		if !strings.HasPrefix(key, "url:") && !strings.HasPrefix(key, "alias:") && !strings.HasPrefix(key, "meta:") {
			err = appendUrl(bkt, key, string(value), false, true)
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = store.Update("meta:v2Migration", []byte("true"))
	return nil
}

func pickReply(replies []*v1.Reply) *v1.Reply {
	if len(replies) == 0 {
		return nil
	}

	if len(replies) == 1 {
		return replies[0]
	}

	return replies[rand.Intn(len(replies))]
}

func gifCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			text := strings.TrimPrefix(cmdMsg.Command.Text, "url:")
			if text != "" {
				var gifUrl string
				var err error
				err = cmdMsg.Store.Get(string(getAliasName(text)), func(v []byte) error {
					var alias *v1.Alias
					if v != nil {
						alias = &v1.Alias{}
						err = proto.Unmarshal(v, alias)
						if err != nil {
							return err
						}
					}

					if alias != nil {
						if len(alias.Allowed) > 0 {
							reply := pickReply(alias.Allowed)
							if reply != nil {
								cmdMsg.Command.Reply() <- &quadlek.CommandResp{
									Text:      reply.Url,
									InChannel: true,
								}
							}
							return nil
						}
					}

					attempts := 0
					for attempts < 10 {
						attempts++
						gUrl, err := gifs.Translate(text)
						if err != nil {
							cmdMsg.Command.Reply() <- &quadlek.CommandResp{
								Text:      fmt.Sprintf("an error occured: %s", err.Error()),
								InChannel: false,
							}
							return nil
						}

						if alias != nil {
							if replyMatch(gUrl, alias.Blocked) {
								continue
							}
						}

						gifUrl = gUrl
						break
					}

					if gifUrl != "" {
						cmdMsg.Command.Reply() <- &quadlek.CommandResp{
							Text:      gifUrl,
							InChannel: true,
						}
					} else {
						cmdMsg.Command.Reply() <- &quadlek.CommandResp{
							Text:      "unable to load unblocked gif url",
							InChannel: false,
						}
					}

					return nil
				})
				if err != nil {
					continue
				}

				if gifUrl != "" {
					err = cmdMsg.Store.Update(fmt.Sprintf("url:%s", gifUrl), []byte(text))
					if err != nil {
						zap.L().Error("error updating store with gif url", zap.Error(err))
					}
				}

			}

		case <-ctx.Done():
			return
		}
	}
}

func gifSaveCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			text := cmdMsg.Command.Text
			parts := strings.Split(text, " ")
			if len(parts) < 2 {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      "Malformed command: /gsave <url> phrase to save",
					InChannel: false,
				}
				continue
			}

			gUrl, err := url.Parse(parts[0])
			if err != nil {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      fmt.Sprintf("Invalid url: %s", parts[1]),
					InChannel: false,
				}
				continue
			}

			phrase := strings.Join(parts[1:], " ")

			err = cmdMsg.Store.UpdateRaw(func(bkt *bolt.Bucket) error {
				return appendUrl(bkt, phrase, gUrl.String(), false, false)
			})
			if err != nil {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      fmt.Sprintf("Unable to save gif phrase: %s", err.Error()),
					InChannel: false,
				}
				continue
			}

			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text:      "Successfully stored gif phrase.",
				InChannel: false,
			}

		case <-ctx.Done():
			return
		}
	}
}

func gifListCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			sb := &strings.Builder{}
			err := cmdMsg.Store.ForEach(func(_ *bolt.Bucket, key string, value []byte) error {
				if len(value) == 0 {
					return nil
				}

				if !strings.HasPrefix(key, "alias:") {
					return nil
				}

				a, err := parseAlias(value)
				if err != nil {
					return err
				}

				fmt.Fprintf(sb, "%s =>\n", a.Phrase)
				if len(a.Allowed) > 0 {
					fmt.Fprintf(sb, "\tAllowed:\n")
					for _, r := range a.Allowed {
						fmt.Fprintf(sb, "\t\t%s\n", r.Url)
					}
				}
				if len(a.Blocked) > 0 {
					fmt.Fprintf(sb, "\tBlocked:\n")
					for _, r := range a.Blocked {
						fmt.Fprintf(sb, "\t\t%s\n", r.Url)
					}
				}
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				continue
			}

			if sb.Len() > 0 {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      sb.String(),
					InChannel: false,
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func gifReaction(ctx context.Context, reactionChannel <-chan *quadlek.ReactionHookMsg) {
	for {
		select {
		case rh := <-reactionChannel:
			switch rh.Reaction.Reaction {
			case GoodBotReaction:
				msg, err := rh.Bot.GetMessage(rh.Reaction.Item.Channel, rh.Reaction.Item.Timestamp)
				if err != nil {
					fmt.Println("error getting message:", err.Error())
					continue
				}

				if msg.User != "" {
					continue
				}

				gifUrl := strings.TrimPrefix(msg.Text, "<")
				gifUrl = strings.TrimSuffix(gifUrl, ">")

				err = rh.Store.UpdateRaw(func(bkt *bolt.Bucket) error {
					b := bkt.Get([]byte(fmt.Sprintf("url:%s", gifUrl)))
					if b == nil {
						return nil
					}

					err = appendUrl(bkt, string(b), gifUrl, false, false)
					if err != nil {
						return err
					}

					return nil
				})
				if err != nil {
					rh.Bot.Say(rh.Reaction.Item.Channel, "Error saving gif")
					continue
				}

			case BadBotReaction:
				msg, err := rh.Bot.GetMessage(rh.Reaction.Item.Channel, rh.Reaction.Item.Timestamp)
				if err != nil {
					fmt.Println("error getting message:", err.Error())
					continue
				}

				if msg.User != "" {
					continue
				}

				gifUrl := strings.TrimPrefix(msg.Text, "<")
				gifUrl = strings.TrimSuffix(gifUrl, ">")

				err = rh.Store.UpdateRaw(func(bkt *bolt.Bucket) error {
					b := bkt.Get([]byte(fmt.Sprintf("url:%s", gifUrl)))
					if b == nil {
						return nil
					}

					err = appendUrl(bkt, string(b), gifUrl, true, false)
					if err != nil {
						return err
					}

					return nil
				})
				if err != nil {
					rh.Bot.Say(rh.Reaction.Item.Channel, "Error saving gif")
					continue
				}
			}

		case <-ctx.Done():
			fmt.Println("Shutting down gif react hook")
			return
		}
	}
}

func Register(apiKey string) quadlek.Plugin {
	gifs = NewGifs(apiKey, "R")
	return quadlek.MakePlugin(
		"gifs",
		[]quadlek.Command{
			quadlek.MakeCommand("g", gifCommand),
			quadlek.MakeCommand("gsave", gifSaveCommand),
			quadlek.MakeCommand("glist", gifListCommand),
		},
		nil,
		[]quadlek.ReactionHook{
			quadlek.MakeReactionHook(gifReaction),
		},
		nil,
		gifLoad,
	)
}
