package main

import (
	"encoding/json"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/google/go-github/github"
)

type MessageHandler struct {
	Repo  *Repository
	Store *blobStore
}

func NewMessageHandler(client *github.Client, config *Config, repo *Repository) *MessageHandler {
	return &MessageHandler{
		Repo: repo,
		Store: &blobStore{
			Client: client,
			Config: config,
		},
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
	b, err := NewBlobFromPayload(event, payload)
	if err = m.Store.PrepareForStorage(m.Repo, b); err != nil {
		log.Errorf("preparing event %q for storage: %v", event, err)
		return err
	}
	return m.Store.Index(StoreLiveEvent, m.Repo, b, delivery)
}
