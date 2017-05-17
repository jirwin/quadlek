package main

import (
	"fmt"

	"gopkg.in/urfave/cli.v2"
)

// Search for a series name
func Search(c *cli.Context) error {
	var query string = c.Args().First()
	if len(query) == 0 {
		return fmt.Errorf("Search parameter \"%s\" is required", query)
	}

	results, err := client.Search.ByName(query)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "  n, Series Name, Series ID\n")
	for n, item := range results {
		fmt.Fprintf(c.App.Writer, "%3d, %s, %d\n", n, item.SeriesName, item.ID)
	}
	return nil
}
