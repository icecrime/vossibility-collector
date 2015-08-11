package main

import (
	"encoding/json"
	"sync"
	"time"

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
		cli.IntFlag{Name: "sleep", Value: 0, Usage: "sleep delay between each GitHub page queried"},
	},
}

const (
	// DefaultFrom is the default starting number for syncing repository items.
	DefaultFrom = 1

	// DefaultNumFetchProcs is the default number of goroutines fetching data
	// from the Github API in parallel.
	DefaultNumFetchProcs = 20

	// DefaultNumIndexProcs is the default number of goroutines indexing data
	// into Elastic Search in parallel.
	DefaultNumIndexProcs = 5

	// DefaultPerPage is the default number of items per page in Github API
	// requests.
	DefaultPerPage = 100

	// DefaultSleepPerPage is the default number of seconds to sleep between
	// each GitHub page queried.
	DefaultSleepPerPage = 0

	// DefaultStorage is the default destination store for the synchronization
	// job.
	DefaultStorage = StoreSnapshot
)

// DefaultSyncOptions is the default set of options for a synchronization job.
var DefaultSyncOptions = syncOptions{
	From:          DefaultFrom,
	NumFetchProcs: DefaultNumFetchProcs,
	NumIndexProcs: DefaultNumIndexProcs,
	PerPage:       DefaultPerPage,
	SleepPerPage:  DefaultSleepPerPage,
	Storage:       DefaultStorage,
}

// doSyncCommand runs a synchronization job: it fetches all GitHub issues and
// pull requests starting with the From index. It uses the API pagination to
// reduce API calls, and allows a Sleep delay between each page to avoid
// triggering the abuse detection mechanism.
func doSyncCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	client := NewClient(config)
	blobStore := NewTransformingBlobStore(config.Transformations)

	// Get the list of repositories.
	repos := make([]*Repository, 0, len(config.Repositories))
	for _, r := range config.Repositories {
		repos = append(repos, r)
	}

	// Create and run the synchronization job.
	cmd := NewSyncCommandWithOptions(client, blobStore, &syncOptions{
		From:          c.Int("from"),
		NumFetchProcs: DefaultNumFetchProcs,
		NumIndexProcs: DefaultNumIndexProcs,
		PerPage:       DefaultPerPage,
		SleepPerPage:  c.Int("sleep"),
		Storage:       DefaultStorage,
	})
	cmd.Run(repos)
}

// syncCmd is a synchronization job.
type syncCmd struct {
	blobStore blobStore
	client    *github.Client
	options   *syncOptions
	toFetch   chan github.Issue
	toIndex   chan githubIndexedItem
	wgFetch   sync.WaitGroup
	wgIndex   sync.WaitGroup
}

// syncOptions is the set of options that can be configured for a
// synchronization job.
type syncOptions struct {
	// From is the index to start syncing from. It can be useful for enormous
	// repositories such as docker/docker to ignore anything past a certain
	// number.
	From int

	// NumFetchProcs is the number of goroutines fetching GitHub data in
	// parallel.
	NumFetchProcs int

	// NumIndexProcs is the number of goroutines storing data into the Elastic
	// Search backend in parallel.
	NumIndexProcs int

	// PerPage is the number of GitHub items to query per page.
	PerPage int

	// SleepPerPage is the number of seconds to sleep between each page queried
	// to avoid triggering GitHub's abuse detection mechanism.
	SleepPerPage int

	// Storage is the type of Storage to Index into.
	Storage Storage
}

// NewSyncCommand creates a default configured synchronization job.
func NewSyncCommand(client *github.Client, blobStore blobStore) *syncCmd {
	return NewSyncCommandWithOptions(client, blobStore, &DefaultSyncOptions)
}

// NewSyncCommandWithOptions creates a synchronization job with the specific
// options set. Be careful when using that function to give meaningful values
// to all options: it is recommand to start from a copy of DefaultSyncOptions
// and modify what needs to be from there.
func NewSyncCommandWithOptions(client *github.Client, blobStore blobStore, opt *syncOptions) *syncCmd {
	return &syncCmd{
		blobStore: blobStore,
		client:    client,
		options:   opt,
		toFetch:   make(chan github.Issue, opt.NumFetchProcs),
		toIndex:   make(chan githubIndexedItem, opt.NumIndexProcs),
	}
}

// Run the synchronization job on the specified repositories. The command From
// options overrides any per-repository starting index.
//
// This function starts NumIndexProcs indexing goroutines and NumFetchProcs
// fetching goroutines, but won't return until all job is done, or a fatal
// error occurs.
//
// Isolated errors (failure to retrieve a particular item, or failure to write
// to the backend) will not interrupt the job. Only the inability to list items
// from GitHub can interrupt prematurely (such as in case of rate limiting).
func (s *syncCmd) Run(repos []*Repository) {
	for _, r := range repos {
		for i := 0; i != s.options.NumIndexProcs; i++ {
			s.wgIndex.Add(1)
			go s.indexingProc(r)
		}

		for i := 0; i != s.options.NumFetchProcs; i++ {
			s.wgFetch.Add(1)
			go s.fetchingProc(r)
		}

		// The command line `--from` option override the configuration defined
		// repository settings.
		from := s.options.From
		if from == 0 {
			from = r.RepositoryConfig.StartIndex
		}
		if err := s.fetchRepositoryItems(r, from, s.options.SleepPerPage); err != nil {
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

// fetchRepositoryItems queries the GitHub API for all issues and pull requests
// for a repository. Any failure to fetch a page interrupts the process and
// returns the error.
//
// The function lists all issues for the repository, and outputs in one of the
// two job channels depending on the nature of the issue. Issues which are
// effective issues are directly sent to the toIndex channel to be stored into
// the Elastic Search backend. Issues which are effectively pull requests get
// sent to the toFetch channel to be enriched by the fetchingProc before being
// stored.
//
// The motivation behind this design is that issues hold a part of the data,
// some of which pull requests don't (in particular labels), but we still need
// the information that are held by the pull request itself (such as additions
// and deletions).
func (s *syncCmd) fetchRepositoryItems(r *Repository, from, sleepPerPage int) error {
	count := 0
	for page := from/s.options.PerPage + 1; page != 0; {
		iss, resp, err := s.client.Issues.ListByRepo(r.User, r.Repo, &github.IssueListByRepoOptions{
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
				s.toIndex <- githubIssue(i)
			} else {
				s.toFetch <- i
			}
		}

		page = resp.NextPage
		if sleepPerPage > 0 {
			time.Sleep(time.Duration(sleepPerPage) * time.Second)
		}
	}
	return nil
}

// fetchingProc takes input from the toFetch channel and fetches additional
// data for items were applicable. In particular, it gets the pull request
// information for issues which are indeed pull requests.
func (s *syncCmd) fetchingProc(r *Repository) {
	for i := range s.toFetch {
		log.Debugf("fetching associated pull request for issue %d", *i.Number)
		if item, err := pullRequestFromIssue(s.client, r, &i); err == nil {
			s.toIndex <- item
		} else {
			s.toIndex <- githubIssue(i)
			log.Errorf("fail to retrieve pull request information for %d: %v", *i.Number, err)
		}
	}
	s.wgFetch.Done()
}

// indexingProc takes input from the toIndex channel and pushes the content to
// the Elastic Search backend.
func (s *syncCmd) indexingProc(r *Repository) {
	for i := range s.toIndex {
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
		if err := s.blobStore.Index(StoreSnapshot, r, b, i.Id()); err != nil {
			log.Error(err)
		}
	}
	s.wgIndex.Done()
}
