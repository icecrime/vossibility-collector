package main

import (
	"strconv"

	"github.com/google/go-github/github"
)

const (
	GithubTypeIssue         = "issue"
	GithubTypePullRequest   = "pull_request"
	SnapshotIssueType       = "snapshot_issue"
	SnapshotPullRequestType = "snapshot_pull_request"
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
	ID() string
	Type() string
}

type githubPR github.PullRequest

func (g githubPR) ID() string {
	return strconv.Itoa(*g.Number)
}

func (g githubPR) Type() string {
	return GithubTypePullRequest
}

type githubIssue github.Issue

func (g githubIssue) ID() string {
	return strconv.Itoa(*g.Number)
}

func (g githubIssue) Type() string {
	return GithubTypeIssue
}

type githubEnrichedPR struct {
	*github.PullRequest
	Labels []github.Label `json:"labels,omitempty"`
}

func (g *githubEnrichedPR) ID() string {
	return strconv.Itoa(*g.PullRequest.Number)
}

func (g *githubEnrichedPR) Type() string {
	return GithubTypePullRequest
}

func pullRequestFromIssue(cli *github.Client, repo *Repository, i *github.Issue) (githubIndexedItem, error) {
	pr, _, err := cli.PullRequests.Get(repo.User, repo.Repo, *i.Number)
	if err != nil {
		return nil, err
	}
	return &githubEnrichedPR{
		PullRequest: pr,
		Labels:      i.Labels,
	}, nil
}
