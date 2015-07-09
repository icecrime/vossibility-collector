package main

import (
	"encoding/json"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/mattbaird/elastigo/core"
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

func onMessage(c *Config, repo *Repository, m *nsq.Message) error {
	// Deserialize body as a partialMessage.
	var p partialMessage
	if err := json.Unmarshal(m.Body, &p); err != nil {
		log.Error(err)
		return err
	}

	return HandleMessage(c, repo, p.GithubEvent, m.Body)
}

func HandleMessage(c *Config, repo *Repository, event string, payload json.RawMessage) error {
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
