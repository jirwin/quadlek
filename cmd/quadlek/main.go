package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	xpost "github.com/jirwin/xpost-quadlek/pkg"
	"go.uber.org/zap"

	"github.com/Bullpeen/infobot"
	gifs "github.com/jirwin/gifs-quadlek/src"
	"github.com/jirwin/quadlek/plugins/karma"
	"github.com/jirwin/quadlek/plugins/random"
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
		zap.L().Error("error creating bot", zap.Error(err))
		return nil
	}

	err = bot.RegisterPlugin(karma.Register())
	if err != nil {
		fmt.Printf("error registering karma plugin: %s\n", err.Error())
		return nil
	}

	err = bot.RegisterPlugin(random.Register())
	if err != nil {
		fmt.Printf("error registering random plugin: %s\n", err.Error())
		return nil
	}

	err = bot.RegisterPlugin(infobot.Register())
	if err != nil {
		fmt.Printf("error registering infobot plugin: %s\n", err.Error())
		return nil
	}

	// err = bot.RegisterPlugin(echo.Register())
	// if err != nil {
	// 	fmt.Printf("error registering echo plugin: %s\n", err.Error())
	// 	return nil
	// }

	xpostPlugin := xpost.Register()
	err = bot.RegisterPlugin(xpostPlugin)
	if err != nil {
		fmt.Printf("error registering xpost plugin: %s\n", err.Error())
		return err
	}

	gifPlugin := gifs.Register(c.String("giphy-api-key"))
	err = bot.RegisterPlugin(gifPlugin)
	if err != nil {
		fmt.Printf("Error registering gifs plugin: %s\n", err.Error())
		return err
	}

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
			EnvVar: "API_TOKEN",
		},
		cli.StringFlag{
			Name:   "verification-token",
			Usage:  "The slack webhook verification token.",
			EnvVar: "VERIFICATION_TOKEN",
		},
		cli.StringFlag{
			Name:   "db-path",
			Usage:  "The path where the database is stored.",
			Value:  "quadlek.db",
			EnvVar: "DB_PATH",
		},
		cli.StringFlag{
			Name:   "giphy-api-key",
			Usage:  "Giphy API Key",
			EnvVar: "GIPHY_KEY",
		},
	}

	app.Run(os.Args)
}
