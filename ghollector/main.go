package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "ghollector"
	app.Usage = "collect Github repository statistics"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "c, config",
			Value: "config.toml",
			Usage: "configuration file",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output",
		},
	}

	app.Action = runCommand.Action
	app.Commands = []cli.Command{
		runCommand,
		syncCommand,
	}

	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
