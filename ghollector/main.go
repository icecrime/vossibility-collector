package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/mattbaird/elastigo/core"
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
		cli.BoolFlag{
			Name:  "debug-es",
			Usage: "enable debug output for elasticsearch queries",
		},
	}

	app.Action = runCommand.Action
	app.Commands = []cli.Command{
		initCommand,
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
