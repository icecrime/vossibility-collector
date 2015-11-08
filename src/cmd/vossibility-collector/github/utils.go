package github

import (
	"strconv"

	"cmd/vossibility-collector/config"
	"cmd/vossibility-collector/storage"

	"github.com/google/go-github/github"
)

type PartialMessage struct {
	GitHubEvent    string `json:"X-GitHub-Event"`
	GitHubDelivery string `json:"X-GitHub-Delivery"`
	HubSignature   string `json:"X-Hub-Signature"`
}

// githubPagedIndexer abstracts functions that list GitHub objects in a paged
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
	return config.GitHubTypePullRequest
}

type githubIssue github.Issue

func (g githubIssue) ID() string {
	return strconv.Itoa(*g.Number)
}

func (g githubIssue) Type() string {
	return config.GitHubTypeIssue
}

type githubEnrichedPR struct {
	*github.PullRequest
	Labels []github.Label `json:"labels,omitempty"`
}

func (g *githubEnrichedPR) ID() string {
	return strconv.Itoa(*g.PullRequest.Number)
}

func (g *githubEnrichedPR) Type() string {
	return config.GitHubTypePullRequest
}

func pullRequestFromIssue(cli *github.Client, repo *storage.Repository, i *github.Issue) (githubIndexedItem, error) {
	pr, _, err := cli.PullRequests.Get(repo.User, repo.Repo, *i.Number)
	if err != nil {
		return nil, err
	}
	return &githubEnrichedPR{
		PullRequest: pr,
		Labels:      i.Labels,
	}, nil
}
