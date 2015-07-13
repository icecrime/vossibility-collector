package main

import (
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
		//if err := syncRepositoryPullRequests(client, r); err != nil {
		//	log.Errorf("error syncing repository %s pull requests: %v", r.PrettyName(), err)
		//}
	}
}

func syncRepositoryIssues(cli *github.Client, repo *Repository) error {
	return syncRepositoryItems(repo, func(page int) ([]githubIndexedItem, *github.Response, error) {
		return listIssues(cli, repo, page)
	})
}

func syncRepositoryPullRequests(cli *github.Client, repo *Repository) error {
	return syncRepositoryItems(repo, func(page int) ([]githubIndexedItem, *github.Response, error) {
		return listPullRequests(cli, repo, page)
	})
}

func syncRepositoryItems(repo *Repository, indexer githubPagedIndexer) error {
	count := 0
	for page := 1; page != 0; {
		items, resp, err := indexer(page)
		if err != nil {
			return err
		}

		count += len(items)
		log.Infof("retrieved %d items for %s (page %d)", count, repo.PrettyName(), page)
		for _, i := range items {
			log.Debugf("store %s #%s", i.Type(), i.Id())
			if _, err := core.Index(repo.SnapshotIndex(), i.Type(), i.Id(), nil, i); err != nil {
				log.Errorf("store pull request %s data: %v", i.Id(), err)
			}
		}

		page = resp.NextPage
	}
	return nil
}
