package main

import (
	"encoding/json"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
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

/*
type syncCmd struct {
	blobStore *blobStore
	toFetch   chan github.Issue
	toIndex   chan githubIndexedItem
	wgFetch   sync.WaitGroup
	wgIndex   sync.WaitGroup
}

func NewSyncCommand() *syncCommand {
	return &syncCommand{
		blobStore: blobStore,
		toFetch:   make(chan github.Issue, NumFetchProcs),
		toIndex:   make(chan githubIndexedItem, NumIndexProcs),
	}
}

func (s *syncCommand) Run(repos []*Repository) {
	for _, r := range config.Repositories {
		for i := 0; i != NumIndexProcs; i++ {
			s.wgIndex.Add(1)
			go s.indexingProc(r)
		}

		for i := 0; i != NumFetchProcs; i++ {
			s.wgFetch.Add(1)
			go fetchingProc(r)
		}

		// TODO c.Int is wrong
		if err := s.fetchRepositoryItems(c.Int("from")); err != nil {
			log.Errorf("error syncing repository %s issues: %v", r.PrettyName(), err)
		}

		// When fetchRepositoryItems is done, all data to fetch has been queued.
		close(s.toFetch)

		// When the fetchingProc is done, all data to index has been queued.
		s.wgFetch.Wait()
		log.Warn("done fetching Github API data")
		close(s.toIndex)

		// Wait until indexing completes.
		s.wgIndex.Wait()
		log.Warn("done indexing documents in Elastic Search")
	}
}
*/

//---------------------------------------------------------------------------//

func doSyncCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	client := NewClient(config)
	blobStore := &blobStore{
		Client: client,
		Config: config,
	}

	toFetch := make(chan github.Issue, NumFetchProcs)
	toIndex := make(chan githubIndexedItem, NumIndexProcs)

	for _, r := range config.Repositories {
		wgIndex := sync.WaitGroup{}
		for i := 0; i != NumIndexProcs; i++ {
			wgIndex.Add(1)
			go indexingProc(blobStore, r, &wgIndex, toIndex)
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
		log.Warn("done fetching Github API data")
		close(toIndex)

		// Wait until indexing completes.
		wgIndex.Wait()
		log.Warn("done indexing documents in Elastic Search")
	}
}

func indexingProc(blobStore *blobStore, repo *Repository, wg *sync.WaitGroup, toIndex <-chan githubIndexedItem) {
	for i := range toIndex {
		// We have to serialize back to JSON in order to transform the payload
		// as we wish. This could be optimized out if we were to read the raw
		// Github data rather than rely on the typed go-github package.
		payload, err := json.Marshal(i)
		if err != nil {
			log.Errorf("error marshaling githubIndexedItem %q (%s): %v", i.Id(), i.Type(), err)
			continue
		}
		// We create a blob from the payload, which essentially deserialized
		// the object back from JSON...
		b, err := NewBlobFromPayload(i.Type(), payload)
		if err != nil {
			log.Errorf("creating blob from payload %q (%s): %v", i.Id(), i.Type(), err)
			continue
		}
		// Persist the object in Elastic Search.
		if err := blobStore.Index(StoreSnapshot, repo, b, ""); err != nil {
			log.Error(err)
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
