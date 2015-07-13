package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-simplejson"
	"github.com/google/go-github/github"
	"github.com/mattbaird/elastigo/core"
	"golang.org/x/oauth2"
)

const (
	GithubTypeIssue       = "issue"
	GithubTypePullRequest = "pull_request"
)

var (
	// GithubEventTypes is the set of all possible Github webhooks events.
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

	// GithubSnapshotedEvents is a map of events for which we want to persist
	// the latest version as a snapshot, associated with the identifier of the
	// payload in the event message.
	GithubSnapshotedEvents = map[string]string{
		"issues":       "issue",
		"pull_request": "pull_request",
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

func handleGithubEvent(c *Config, repo *Repository, event, delivery string, payload json.RawMessage) error {
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

	// Store the event in Elastic Search: index is determined by the repository
	// and type by the event type.
	if err := storeGithubEvent(repo, event, delivery, data); err != nil {
		return err
	}
	if err := storeGithubSnapshot(repo, event, delivery, data); err != nil {
		return err
	}
	return nil
}

func storeGithubEvent(repo *Repository, event, delivery string, data []byte) error {
	if _, err := core.Index(repo.EventsIndex(), event, delivery, nil, data); err != nil {
		return err
	}
	return nil
}

func storeGithubSnapshot(repo *Repository, event, delivery string, data []byte) error {
	payloadField, ok := GithubSnapshotedEvents[event]
	if !ok {
		return nil
	}

	// Specifically extract the payload field from the JSON body.
	sj, err := simplejson.NewJson(data)
	if err != nil {
		return err
	}
	sp := sj.Get(payloadField)
	payload, err := sp.MarshalJSON()
	if err != nil {
		return err
	}

	//  We assume that any type of snapshoted event has a "number" attribute.
	payloadId := strconv.Itoa(sp.Get("number").MustInt())
	if _, err := core.Index(repo.SnapshotIndex(), payloadField, payloadId, nil, payload); err != nil {
		return err
	}
	return nil
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
