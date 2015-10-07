package main

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var syncCommand = cli.Command{
	Name:   "sync",
	Usage:  "sync storage with the GitHub repositories",
	Action: doSyncCommand,
	Flags: []cli.Flag{
		cli.IntFlag{Name: "from", Value: 1, Usage: "issue number to start from"},
		cli.IntFlag{Name: "sleep", Value: 0, Usage: "sleep delay between each GitHub page queried"},
	},
}

// doSyncCommand runs a synchronization job: it fetches all GitHub issues and
// pull requests starting with the From index. It uses the API pagination to
// reduce API calls, and allows a Sleep delay between each page to avoid
// triggering the abuse detection mechanism.
func doSyncCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	client := NewClient(config)
	blobStore := NewTransformingBlobStore()

	// Get the list of repositories from command-line (defaults to all).
	repoToSync := c.Args()
	if len(repoToSync) == 0 {
		repoToSync = make([]string, 0, len(config.Repositories))
		for givenName, _ := range config.Repositories {
			repoToSync = append(repoToSync, givenName)
		}
	}

	// Get the repositories instances from their given names.
	repos := make([]*Repository, 0, len(repoToSync))
	for _, givenName := range repoToSync {
		r, ok := config.Repositories[givenName]
		if !ok {
			log.Fatalf("unknown repository %q", givenName)
		}
		repos = append(repos, r)
	}

	// Configure a syncJob taking all issues (opened and closed) and storing
	// in the snapshot store.
	syncOptions := DefaultSyncOptions
	syncOptions.From = c.Int("from")
	syncOptions.SleepPerPage = c.Int("sleep")
	syncOptions.State = GitHubStateFilterAll
	syncOptions.Storage = StoreSnapshot

	// Create and run the synchronization job.
	log.Warnf("running sync jobs on repositories %s", strings.Join(repoToSync, ", "))
	NewSyncCommandWithOptions(client, blobStore, &syncOptions).Run(repos)
}
