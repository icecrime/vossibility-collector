package main

import (
	"math"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
	"github.com/mattbaird/elastigo/core"
)

var syncCommand = cli.Command{
	Name:   "sync",
	Usage:  "sync storage with the Github repositories",
	Action: doSyncCommand,
}

func doSyncCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	client := newGithubClient(config)
	for _, r := range config.GetRepositories() {
		if err := syncRepositoryIssues(client, r); err != nil {
			log.Errorf("error syncing repository %s issues: %v", r.PrettyName(), err)
		}
		if err := syncRepositoryPullRequests(client, r); err != nil {
			log.Errorf("error syncing repository %s pull requests: %v", r.PrettyName(), err)
		}
	}
}

func syncRepositoryIssues(cli *github.Client, repo *Repository) error {
	return syncRepositoryItems(repo, "issue", func(page int) ([]githubIndexedItem, *github.Response, error) {
		return listIssues(cli, repo, page)
	})
}

func syncRepositoryPullRequests(cli *github.Client, repo *Repository) error {
	return syncRepositoryItems(repo, "pull", func(page int) ([]githubIndexedItem, *github.Response, error) {
		return listPullRequests(cli, repo, page)
	})
}

func syncRepositoryItems(repo *Repository, itemType string, indexer githubPagedIndexer) error {
	count := 0
	final := math.MaxInt32
	for page := 1; page < final; page++ {
		time.Sleep(1 * time.Second)
		prs, resp, err := indexer(page)
		if err != nil {
			return err
		}
		final = resp.LastPage
		count += len(prs)

		log.Debugf("retrieved %d %ss for %s (page %d)", count, itemType, repo.PrettyName(), page)
		for _, pr := range prs {
			if _, err := core.Index(repo.SnapshotIndex(), itemType, pr.Id(), nil, pr); err != nil {
				log.Errorf("store pull request %s data: %v", pr.Id(), err)
			}
		}
	}
	return nil
}
