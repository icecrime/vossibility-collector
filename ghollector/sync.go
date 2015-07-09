package main

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/oauth2"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
	"github.com/mattbaird/elastigo/api"
	"github.com/mattbaird/elastigo/core"
)

var syncCommand = cli.Command{
	Name:   "sync",
	Usage:  "sync storage with the Github repositories",
	Action: doSyncCommand,
}

func doSyncCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	api.Domain = config.ElasticSearch

	// TODO Factor out
	var tc *http.Client
	if config.GithubApiToken != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: config.GithubApiToken,
		})
		tc = oauth2.NewClient(oauth2.NoContext, ts)
	}
	client := github.NewClient(tc)

	// Update the current state of every pull request.
	for _, r := range config.GetRepositories() {
		if err := syncRepositoryIssues(client, r); err != nil {
			log.Errorf("error syncing repository %s issues: %v", r.PrettyName(), err)
		}
		if err := syncRepositoryPullRequests(client, r); err != nil {
			log.Errorf("error syncing repository %s pull requests: %v", r.PrettyName(), err)
		}
	}
}

type GithubPagedIndexer func(page int) ([]GithubIndexedItem, *github.Response, error)

type GithubIndexedItem interface {
	Id() string
}

type GithubPR github.PullRequest

func (g GithubPR) Id() string {
	return strconv.Itoa(*g.Number)
}

type GithubIssue github.Issue

func (g GithubIssue) Id() string {
	return strconv.Itoa(*g.Number)
}

func syncRepositoryIssues(cli *github.Client, repo *Repository) error {
	return syncRepositoryItems(repo, func(page int) ([]GithubIndexedItem, *github.Response, error) {
		return listIssues(cli, repo, page)
	})
}

func syncRepositoryPullRequests(cli *github.Client, repo *Repository) error {
	return syncRepositoryItems(repo, func(page int) ([]GithubIndexedItem, *github.Response, error) {
		return listPullRequests(cli, repo, page)
	})
}

func syncRepositoryItems(repo *Repository, indexer GithubPagedIndexer) error {
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

		log.Debugf("retrieved %d pull requests for %s (page %d)", count, repo.PrettyName(), page)
		for _, pr := range prs {
			if _, err := core.Index(repo.LatestIndex(), "issue", pr.Id(), nil, pr); err != nil {
				log.Errorf("store pull request %s data: %v", pr.Id(), err)
			}
		}
	}
	return nil
}

func listIssues(cli *github.Client, repo *Repository, page int) ([]GithubIndexedItem, *github.Response, error) {
	iss, resp, err := cli.Issues.ListByRepo(repo.User, repo.Repo, &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: 100,
		},
	})

	out := make([]GithubIndexedItem, 0, len(iss))
	for _, i := range iss {
		out = append(out, GithubIssue(i))
	}
	return out, resp, err
}

func listPullRequests(cli *github.Client, repo *Repository, page int) ([]GithubIndexedItem, *github.Response, error) {
	prs, resp, err := cli.PullRequests.List(repo.User, repo.Repo, &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: 100,
		},
	})

	out := make([]GithubIndexedItem, 0, len(prs))
	for _, p := range prs {
		out = append(out, GithubPR(p))
	}
	return out, resp, err
}
