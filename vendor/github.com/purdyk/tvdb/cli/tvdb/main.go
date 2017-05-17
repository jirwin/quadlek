package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/purdyk/tvdb"
	"gopkg.in/urfave/cli.v2"
)

var baseHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
}
var client *tvdb.Client

func main() {
	app := cli.NewApp()
	app.Name = "tvdb"
	app.Usage = "query thetvdb.com database"
	app.Version = "0.1.0"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "apikey, k",
			Usage:  "set the service `APIKEY`",
			EnvVar: "APIKEY",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "search",
			Aliases: []string{"s"},
			Action:  Search,
		},
		{
			Name:   "series",
			Action: Series,
		},
		{
			Name: "next",
			Aliases: []string{"n"},
			Action: Next,
		},
	}

	app.Before = func(c *cli.Context) error {
		apiKey := c.String("apikey")
		if len(apiKey) <= 0 {
			return fmt.Errorf("API Key required")
		}

		auth := new(tvdb.Auth)
		auth.APIKey = apiKey
		client = tvdb.NewClient(baseHTTPClient, auth)
		_, err := client.Token.Login()
		return err
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(app.Writer, "Error", err)
	}
}
