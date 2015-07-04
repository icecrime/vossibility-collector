package main

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
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
	Payload        json.RawMessage
}

func isSubscribed(c *Config, repo string, event string) bool {
	e, _ := c.GetRepositoryEventSet(repo)
	fmt.Printf("EventSet for %q = %v\n", repo, e)
	return e.Contains(event)
}

func isValidEventType(event string) bool {
	return GithubEventTypes[event]
}

func onMessage(c *Config, repo string, m *nsq.Message) error {
	// Deserialize body as a partialMessage.
	var p partialMessage
	if err := json.Unmarshal(m.Body, &p); err != nil {
		log.Error(err)
		return err
	}

	// Check if we are subscribed to this particular event type.
	if !isSubscribed(c, repo, p.GithubEvent) {
		log.Infof("Ignoring event %q for repository %q", p.GithubEvent, repo)
		return nil
	}

	log.Infof("Receive event %q for repository %q", p.GithubEvent, repo)
	return HandleMessage(c, p.GithubEvent, p.Payload)
}

func HandleMessage(c *Config, event string, payload json.RawMessage) error {
	data, err := c.Masks[event].Apply(payload)
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", data)
	return nil
}
