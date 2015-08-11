package main

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/google/go-github/github"
)

func liveEventType(event string) string {
	return fmt.Sprintf("%s_event", event)
}

type MessageHandler struct {
	Client *github.Client
	Repo   *Repository
	Store  blobStore
}

func NewMessageHandler(client *github.Client, config *Config, repo *Repository) *MessageHandler {
	return &MessageHandler{
		Client: client,
		Repo:   repo,
		Store:  NewTransformingBlobStore(config.Transformations),
	}
}

func (m *MessageHandler) HandleMessage(n *nsq.Message) error {
	var p partialMessage
	if err := json.Unmarshal(n.Body, &p); err != nil {
		log.Error(err)
		return nil // No need to retry
	}
	return m.handleEvent(p.GithubEvent, p.GithubDelivery, n.Body)
}

func (m *MessageHandler) handleEvent(event, delivery string, payload json.RawMessage) error {
	// Check if we are subscribed to this particular event type.
	if !m.Repo.IsSubscribed(event) {
		log.Debugf("ignoring event %q for repository %s", event, m.Repo.PrettyName())
		return nil
	}
	log.Infof("receive event %q for repository %q", event, m.Repo.PrettyName())

	// Create the blob object and complete any data that needs to be.
	b, err := NewBlobFromPayload(liveEventType(event), payload)
	if err = m.prepareForStorage(b); err != nil {
		log.Errorf("preparing event %q for storage: %v", event, err)
		return err
	}
	return m.Store.Index(StoreLiveEvent, m.Repo, b, delivery)
}

func (m *MessageHandler) prepareForStorage(o *Blob) error {
	if o.Type() == EvtPullRequest && !o.HasAttribute(LabelsAttribute) {
		number := o.Data.Get("number").MustInt()
		log.Debugf("fetching labels for %s #%d", m.Repo.PrettyName(), number)
		l, _, err := m.Client.Issues.ListLabelsByIssue(m.Repo.User, m.Repo.Repo, number, &github.ListOptions{})
		if err != nil {
			return fmt.Errorf("retrieve labels for issue %s: %v", number, err)
		}
		o.Push(LabelsAttribute, l)
	}
	return nil
}
