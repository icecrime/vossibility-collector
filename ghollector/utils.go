package main

import (
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
)

const (
	GithubTypeIssue       = "issue"
	GithubTypePullRequest = "pull_request"
)

type partialMessage struct {
	GithubEvent    string `json:"X-Github-Event"`
	GithubDelivery string `json:"X-Github-Delivery"`
	HubSignature   string `json:"X-Hub-Signature"`
}

// githubPagedIndexer abstracts functions that list Github objects in a paged
// manner such as issues and pull requests.
type githubPagedIndexer func(page int) ([]githubIndexedItem, *github.Response, error)

type githubIndexedItem interface {
	Id() string
	Type() string
}

type githubPR github.PullRequest

func (g githubPR) Id() string {
	return strconv.Itoa(*g.Number)
}

func (g githubPR) Type() string {
	return GithubTypePullRequest
}

type githubIssue github.Issue

func (g githubIssue) Id() string {
	return strconv.Itoa(*g.Number)
}

func (g githubIssue) Type() string {
	return GithubTypeIssue
}

type githubEnrichedPR struct {
	*github.PullRequest
	Labels []github.Label `json:"labels,omitempty"`
}

func (g *githubEnrichedPR) Id() string {
	return strconv.Itoa(*g.PullRequest.Number)
}

func (g *githubEnrichedPR) Type() string {
	return GithubTypePullRequest
}

func pullRequestFromIssue(cli *github.Client, repo *Repository, i *github.Issue) (githubIndexedItem, error) {
	log.Debugf("fetching associated pull request for issue %d", *i.Number)
	pr, _, err := cli.PullRequests.Get(repo.User, repo.Repo, *i.Number)
	if err != nil {
		return nil, err
	}
	return &githubEnrichedPR{
		PullRequest: pr,
		Labels:      i.Labels,
	}, nil
}

func listIssues(cli *github.Client, repo *Repository, page int) ([]githubIndexedItem, *github.Response, error) {
	iss, resp, err := cli.Issues.ListByRepo(repo.User, repo.Repo, &github.IssueListByRepoOptions{
		Direction: "asc", // List by created date ascending
		Sort:      "created",
		State:     "all", // Include closed issues
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: 100,
		},
	})

	// If the issue is really a pull request, fetch it as such and merge the
	// two objects.
	out := make([]githubIndexedItem, 0, len(iss))
	for _, i := range iss {
		if i.PullRequestLinks == nil {
			out = append(out, githubIssue(i))
		} else if item, err := pullRequestFromIssue(cli, repo, &i); err == nil {
			out = append(out, item)
		} else {
			out = append(out, githubIssue(i))
			log.Errorf("fail to retrieve pull request information for %d: %v", *i.Number, err)
		}
	}
	return out, resp, err
}

func listPullRequests(cli *github.Client, repo *Repository, page int) ([]githubIndexedItem, *github.Response, error) {
	prs, resp, err := cli.PullRequests.List(repo.User, repo.Repo, &github.PullRequestListOptions{
		Direction: "asc", // List by created date ascending
		Sort:      "created",
		State:     "all", // Include closed pull requests
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: 100,
		},
	})

	out := make([]githubIndexedItem, 0, len(prs))
	for _, p := range prs {
		out = append(out, githubPR(p))
	}
	return out, resp, err
}
