package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/mattbaird/elastigo/core"
	"golang.org/x/oauth2"
)

var (
	GithubEventTypes = map[string]bool{
		"pull_request_review_comment": true,
		"commit_comment":              true,
		"create":                      true,
		"delete":                      true,
		"deployment":                  true,
		"deployment_status":           true,
		"follow":                      true,
		"fork":                        true,
		"fork_apply":                  true,
		"gollum":                      true,
		"issue_comment":               true,
		"issues":                      true,
		"member":                      true,
		"membership":                  true,
		"page_build":                  true,
		"public":                      true,
		"pull_request":                true,
		"push":                        true,
		"release":                     true,
		"repositories":                true,
		"status":                      true,
		"team_add":                    true,
		"watch":                       true,
	}
)

type partialMessage struct {
	GithubEvent    string `json:"X-Github-Event"`
	GithubDelivery string `json:"X-Github-Delivery"`
	HubSignature   string `json:"X-Hub-Signature"`
}

func isValidEventType(event string) bool {
	return GithubEventTypes[event]
}

func newGithubClient(config *Config) *github.Client {
	var tc *http.Client
	if config.GithubApiToken != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: config.GithubApiToken,
		})
		tc = oauth2.NewClient(oauth2.NoContext, ts)
	}
	return github.NewClient(tc)
}

func handleGithubEvent(c *Config, repo *Repository, event string, payload json.RawMessage) error {
	// Check if we are subscribed to this particular event type.
	if !repo.IsSubscribed(event) {
		log.Debugf("Ignoring event %q for repository %s", event, repo.PrettyName())
		return nil
	}
	log.Infof("Receive event %q for repository %q", event, repo)

	// We rely on the fact that a defaut constructed Mask is a pass-through, so
	// we don't need to test if it's even defined.
	data, err := c.Masks[event].Apply(payload)
	if err != nil {
		log.Errorf("Failed to apply mask for event %q: %v", event, err)
		return err
	}

	// Store the event in Elastic Search.
	if _, err := core.Index(repo.EventsIndex(), "event", "", nil, data); err != nil {
		return err
	}
	return nil
}

// githubPagedIndexer abstracts functions that list Github objects in a paged
// manner such as issues and pull requests.
type githubPagedIndexer func(page int) ([]githubIndexedItem, *github.Response, error)

type githubIndexedItem interface {
	Id() string
}

type githubPR github.PullRequest

func (g githubPR) Id() string {
	return strconv.Itoa(*g.Number)
}

type githubIssue github.Issue

func (g githubIssue) Id() string {
	return strconv.Itoa(*g.Number)
}

func listIssues(cli *github.Client, repo *Repository, page int) ([]githubIndexedItem, *github.Response, error) {
	iss, resp, err := cli.Issues.ListByRepo(repo.User, repo.Repo, &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: 100,
		},
	})

	out := make([]githubIndexedItem, 0, len(iss))
	for _, i := range iss {
		out = append(out, githubIssue(i))
	}
	return out, resp, err
}

func listPullRequests(cli *github.Client, repo *Repository, page int) ([]githubIndexedItem, *github.Response, error) {
	prs, resp, err := cli.PullRequests.List(repo.User, repo.Repo, &github.PullRequestListOptions{
		State: "all",
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
