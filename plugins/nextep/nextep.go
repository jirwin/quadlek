package nextep

import (
	"context"
	"github.com/jirwin/quadlek/quadlek"
	"github.com/purdyk/tvdb"
	"net/http"
	"time"
	"fmt"
	"errors"
)

var tvdbKey string;

func getTVDBClient(authToken string) (*tvdb.Client) {
	auth := &tvdb.Auth{APIKey: authToken}

	hClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	tClient := tvdb.NewClient(hClient, auth)

	tClient.Token.Login()

	return tClient

}

func findShowId(client *tvdb.Client, name string) (int32, error) {
	results, err := client.Search.ByName(name)

	if err != nil {
		return -1, err
	}

	if len(results) == 0 {
		return -1, errors.New("Found No Results")
	}

	return results[0].ID, nil

}

func findFirstEpisode(client *tvdb.Client, showId int32) (*tvdb.Episode, error) {
	links := &tvdb.Links{}
	links.Next = 1

	params := &tvdb.EpisodeSearchParams{}

	for {
		page := fmt.Sprintf("%d", links.Next)
		params.Page = &page

		results, err := client.Episodes.ListEpisodes(showId, params)

		if err != nil {
			return nil, err
		}

		for _, ep := range results.Data {
			if ep.IsInFuture() {
				return &ep, nil
			}
		}

		links := results.Links

		if !links.HasNext() {
			break
		}
	}

	return nil, errors.New("Failed to find a future episode")

}

func nextEpCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			text := cmdMsg.Command.Text
			client := getTVDBClient(tvdbKey)

			id, err := findShowId(client, text)

			if err != nil {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text: fmt.Sprintf("Show Search Failed: %s", err),
				}
				continue
			}

			series, err := client.Series.Get(id)

			if err != nil {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text: fmt.Sprintf("Series Lookup Failed: %s", err),
				}
				continue
			}

			if series.Status != "Continuing" {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text: fmt.Sprintf("Series has ended"),
				}
				continue
			}

			ep, err := findFirstEpisode(client, id)

			if err != nil {
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text: fmt.Sprintf("Failed to locate first episode: %s", err),
				}
				continue
			}

			cmdMsg.Command.Reply() <- &quadlek.CommandResp{
				Text:      fmt.Sprintf("Next Episode\n\t%s\n\t%s at %s\n", ep.EpisodeName, ep.FirstAired, series.AirsTime),
				InChannel: true,
			}

		case <-ctx.Done():
			return
		}
	}
}

func Register(apikey string) quadlek.Plugin {
	tvdbKey = apikey

	return quadlek.MakePlugin(
		"TVDB",
		[]quadlek.Command{
			quadlek.MakeCommand("nextep", nextEpCommand),
		},
		nil,
		nil,
		nil,
	)
}
