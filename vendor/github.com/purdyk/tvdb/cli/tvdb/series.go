package main

import (
	"fmt"
	"strconv"

	"gopkg.in/urfave/cli.v2"
)

// Series looksup series by ID
func Series(c *cli.Context) error {
	var idString string = c.Args().First()
	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		return err
	}
	if id == 0 {
		return fmt.Errorf("Id parameter is required")
	}

	series, err := client.Series.Get(int32(id))
	if err != nil {
		return err
	}
	if series.ID != int32(id) {
		fmt.Fprintf(c.App.Writer, "Series ID %d could not be found\n", id)
		return nil
	}

	fmt.Fprintf(c.App.Writer, "%s: \n", series.SeriesName)
	fmt.Fprintf(c.App.Writer, "  Rating:   %-2.1f\n", series.SiteRating)
	fmt.Fprintf(c.App.Writer, "  Overview: %s\n\n", series.Overview)
	fmt.Fprintf(c.App.Writer, "  Network:  %s\n", series.Network)
	fmt.Fprintf(c.App.Writer, "  Status:   %s\n", series.Status)
	if series.Status != "Ended" {
		fmt.Fprintf(c.App.Writer, "  Airs:     %s, %s\n", series.AirsDayOfWeek, series.AirsTime)
	}
	return nil
}
