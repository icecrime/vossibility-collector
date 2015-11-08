package main

import (
	"os"
	"text/template"

	"cmd/vossibility-collector/github"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

const OutputFormat = `Core:
  Limit:     {{ .Core.Limit }}
  Remaining: {{ .Core.Remaining }}
  Reset:     {{ .Core.Reset }}

Search:
  Limit:     {{ .Search.Limit }}
  Remaining: {{ .Search.Remaining }}
  Reset:     {{ .Search.Reset }}
`

var limitsCommand = cli.Command{
	Name:   "limits",
	Usage:  "get information about your GitHub API rate limits",
	Action: doLimitsCommand,
}

func doLimitsCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	client := github.NewClient(config.GitHubAPIToken)

	rl, _, err := client.RateLimits()
	if err != nil {
		log.Fatal(err)
	}

	tmpl, _ := template.New("").Parse(OutputFormat)
	tmpl.Execute(os.Stdout, rl)
}
