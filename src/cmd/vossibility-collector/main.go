package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/mattbaird/elastigo/core"
)

func main() {
	app := cli.NewApp()
	app.Name = "vossibility-collector"
	app.Usage = "collect Github repository data"
	app.Version = "0.1.0"

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
		cli.BoolFlag{
			Name:  "debug-es",
			Usage: "enable debug output for elasticsearch queries",
		},
	}

	app.Action = runCommand.Action
	app.Commands = []cli.Command{
		limitsCommand,
		runCommand,
		syncCommand,
		syncMappingCommand,
		syncUsersCommand,
	}

	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		core.VerboseLogging = c.GlobalBool("debug-es")
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
