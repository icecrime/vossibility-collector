package main

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
	"github.com/mattbaird/elastigo/core"
)

var syncCommand = cli.Command{
	Name:   "sync",
	Usage:  "sync storage with the Github repositories",
	Action: doSyncCommand,
	Flags: []cli.Flag{
		cli.IntFlag{Name: "from", Value: 1, Usage: "issue number to start from"},
	},
}

const (
	// NumFetchProcs is the number of goroutines fetching data from the Github
	// API in parallel.
	NumFetchProcs = 20

	// NumIndexProcs is the number of goroutines indexing data into Elastic
	// Search in parallel.
	NumIndexProcs = 5

	// PerPage is the number of items per page in Github API requests.
	PerPage = 100
)

func doSyncCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))

	client := NewClient(config)
	toFetch := make(chan github.Issue, NumFetchProcs)
	toIndex := make(chan githubIndexedItem, NumIndexProcs)

	for _, r := range config.GetRepositories() {
		wgIndex := sync.WaitGroup{}
		for i := 0; i != NumIndexProcs; i++ {
			wgIndex.Add(1)
			go indexingProc(client, r, &wgIndex, toIndex)
		}

		wgFetch := sync.WaitGroup{}
		for i := 0; i != NumFetchProcs; i++ {
			wgFetch.Add(1)
			go fetchingProc(client, r, &wgFetch, toFetch, toIndex)
		}

		if err := fetchRepositoryItems(client, r, toFetch, toIndex, c.Int("from")); err != nil {
			log.Errorf("error syncing repository %s issues: %v", r.PrettyName(), err)
		}

		// When fetchRepositoryItems is done, all data to fetch has been queued.
		close(toFetch)

		// When the fetchingProc is done, all data to index has been queued.
		wgFetch.Wait()
		log.Warn("Done fetching Github API data")
		close(toIndex)

		// Wait until indexing completes.
		wgIndex.Wait()
		log.Warn("Done indexing documents in Elastic Search")
	}
}

func indexingProc(cli *github.Client, repo *Repository, wg *sync.WaitGroup, toIndex <-chan githubIndexedItem) {
	// TODO
	// Go through the same code as live updates
	// Mask the result
	// If labels missing, query the labels endpoint
	for i := range toIndex {
		log.Debugf("store %s #%s", i.Type(), i.Id())
		if _, err := core.Index(repo.SnapshotIndex(), i.Type(), i.Id(), nil, i); err != nil {
			log.Errorf("store pull request %s data: %v", i.Id(), err)
		}
	}
	wg.Done()
}

func fetchingProc(cli *github.Client, repo *Repository, wg *sync.WaitGroup, toFetch <-chan github.Issue, toIndex chan<- githubIndexedItem) {
	for i := range toFetch {
		log.Debugf("fetching associated pull request for issue %d", *i.Number)
		if item, err := pullRequestFromIssue(cli, repo, &i); err == nil {
			toIndex <- item
		} else {
			toIndex <- githubIssue(i)
			log.Errorf("fail to retrieve pull request information for %d: %v", *i.Number, err)
		}
	}
	wg.Done()
}

func fetchRepositoryItems(cli *github.Client, r *Repository, toFetch chan<- github.Issue, toIndex chan<- githubIndexedItem, from int) error {
	count := 0
	for page := from/PerPage + 1; page != 0; {
		iss, resp, err := cli.Issues.ListByRepo(r.User, r.Repo, &github.IssueListByRepoOptions{
			Direction: "asc", // List by created date ascending
			Sort:      "created",
			State:     "all", // Include closed issues
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		})
		if err != nil {
			return err
		}

		count += len(iss)
		log.Infof("retrieved %d items for %s (page %d)", count, r.PrettyName(), page)

		// If the issue is really a pull request, fetch it as such.
		for _, i := range iss {
			if i.PullRequestLinks == nil {
				toIndex <- githubIssue(i)
			} else {
				toFetch <- i
			}
		}

		page = resp.NextPage
	}
	return nil
}
