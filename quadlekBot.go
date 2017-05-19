package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/plugins/echo"
	"github.com/jirwin/quadlek/plugins/karma"
	"github.com/jirwin/quadlek/plugins/nextep"
	"github.com/jirwin/quadlek/plugins/random"
	"github.com/jirwin/quadlek/plugins/spotify"
	"github.com/jirwin/quadlek/plugins/twitter"
	"github.com/jirwin/quadlek/quadlek"
	"github.com/urfave/cli"
)

const Version = "0.0.1"

func run(c *cli.Context) error {
	var apiToken string
	if c.IsSet("api-key") {
		apiToken = c.String("api-key")
	} else {
		cli.ShowAppHelp(c)
		return cli.NewExitError("Missing --api-key arg.", 1)
	}

	var verificationToken string
	if c.IsSet("verification-token") {
		verificationToken = c.String("verification-token")
	} else {
		cli.ShowAppHelp(c)
		return cli.NewExitError("Missing --verification-token arg.", 1)
	}

	dbPath := c.String("db-path")

	bot, err := quadlek.NewBot(context.Background(), apiToken, verificationToken, dbPath)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error creating bot")
		return nil
	}

	err = bot.RegisterPlugin(echo.Register())
	if err != nil {
		fmt.Printf("error registering echo plugin: %s", err.Error())
		return nil
	}

	err = bot.RegisterPlugin(karma.Register())
	if err != nil {
		fmt.Printf("error registering karma plugin: %s", err.Error())
		return nil
	}

	err = bot.RegisterPlugin(random.Register())
	if err != nil {
		fmt.Printf("error registering random plugin: %s", err.Error())
		return nil
	}

	err = bot.RegisterPlugin(spotify.Register())
	if err != nil {
		fmt.Printf("error registering spotify plugin: %s", err.Error())
		return nil
	}

	if c.IsSet("tvdb-key") {
		tvdbKey := c.String("tvdb-key")

		err = bot.RegisterPlugin(nextep.Register(tvdbKey))
		if err != nil {
			fmt.Printf("error registering nextep plugin: %s", err.Error())
			return nil
		}
	}

	err = bot.RegisterPlugin(twitter.Register(
		c.String("twitter-consumer-key"),
		c.String("twitter-consumer-secret"),
		c.String("twitter-access-token"),
		c.String("twitter-access-secret"),
		map[string]string{
			"25073877":           "politics",
			"13658502":           "general",
			"778682":             "general",
			"746780308457009153": "general",
			"26786244":           "general",
			"770397982797602816": "general",
			"2317524115":         "general",
			"807095":             "politics",
			"2467791":            "politics",
		},
	))

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	bot.Start()
	<-signals
	bot.Stop()

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "quadlek"
	app.Version = Version
	app.Usage = "a slack bot"
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "api-key",
			Usage:  "The slack api token for the bot",
			EnvVar: "QUADLEK_API_TOKEN",
		},
		cli.StringFlag{
			Name:   "verification-token",
			Usage:  "The slack webhook verification token.",
			EnvVar: "QUADLEK_VERIFICATION_TOKEN",
		},
		cli.StringFlag{
			Name:   "db-path",
			Usage:  "The path where the database is stored.",
			Value:  "quadlek.db",
			EnvVar: "QUADLEK_DB_PATH",
		},
		cli.StringFlag{
			Name:   "tvdb-key",
			Usage:  "The TVDB api key for the bot, used by the nextep command",
			EnvVar: "QUADLEK_TVDB_TOKEN",
		},
		cli.StringFlag{
			Name:   "twitter-consumer-key",
			Usage:  "The consumer key for the twitter api",
			EnvVar: "QUADLEK_TWITTER_CONSUMER_KEY",
		},
		cli.StringFlag{
			Name:   "twitter-consumer-secret",
			Usage:  "The consumer secret for the twitter api",
			EnvVar: "QUADLEK_TWITTER_CONSUMER_SECRET",
		},
		cli.StringFlag{
			Name:   "twitter-access-token",
			Usage:  "The access key for the twitter api",
			EnvVar: "QUADLEK_TWITTER_ACCESS_TOKEN",
		},
		cli.StringFlag{
			Name:   "twitter-access-secret",
			Usage:  "The access secret for the twitter api",
			EnvVar: "QUADLEK_TWITTER_ACCESS_SECRET",
		},
	}

	app.Run(os.Args)
}
