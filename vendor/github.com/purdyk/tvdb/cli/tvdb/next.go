package main

import (
	"fmt"
	"strconv"

	"gopkg.in/urfave/cli.v2"
	"github.com/purdyk/tvdb"
)

// Finds the next episode of a show by ID
func Next(c *cli.Context) error {
	var idString string = c.Args().First()
	id, err := strconv.ParseInt(idString, 10, 64)

	if err != nil {
		return err
	}
	if id == 0 {
		return fmt.Errorf("Id parameter is required")
	}

	series, err := client.Series.Get(int32(id))

	if series.Status != "Continuing" {
		fmt.Fprintf(c.App.Writer, "Series has ended")
	}

	links := &tvdb.Links{}
	links.Next = 1

	params := &tvdb.EpisodeSearchParams{}

	for {
		page := fmt.Sprintf("%d", links.Next)
		params.Page = &page

		results, err := client.Episodes.ListEpisodes(int32(id), params)

		if err != nil {
			return err
		}

		for _, ep := range results.Data {
			if (ep.IsInFuture()) {
				fmt.Fprintf(c.App.Writer, "Next Episode\n\t%s\n\t%s at %s\n", ep.EpisodeName, ep.FirstAired, series.AirsTime)
				return nil
			}
		}

		links := results.Links

		if !links.HasNext() {
			break
		}
	}

	fmt.Fprintf(c.App.Writer, "Couldn't find a next episode")

	return nil
}
