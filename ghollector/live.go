package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/bitly/go-simplejson"
	"github.com/google/go-github/github"
	"github.com/mattbaird/elastigo/core"
)

type MessageHandler struct {
	Client *github.Client
	Config *Config
	Repo   *Repository
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
		log.Debugf("Ignoring event %q for repository %s", event, m.Repo.PrettyName())
		return nil
	}
	log.Infof("Receive event %q for repository %q", event, m.Repo.PrettyName())

	// We rely on the fact that a defaut constructed Mask is a pass-through, so
	// we don't need to test if it's even defined.
	trans := m.Config.Transformations[event]

	b, err := NewBlobFromPayload(event, payload)
	if b, err = PrepareForStorage(m.Client, m.Repo, b, trans); err != nil {
		log.Errorf("Failed to apply transformation for event %q: %v", event, err)
		return err
	}

	// Store the event in Elastic Search: index is determined by the repository
	// and type by the event type (potential overriden by metadata).
	data, err := b.Encode()
	if err != nil {
		log.Errorf("Failed to serialize transformed result: %v", err)
	}

	fmt.Printf("Push = %s\n", string(data))

	// TODO Store in the last hourly snapshot
	if err := storeGithubEvent(m.Repo, event, delivery, b.Type(), data); err != nil {
		return err
	}
	if err := storeGithubSnapshot(m.Repo, event, delivery, b.Type(), data); err != nil {
		return err
	}
	return nil
}

func storeGithubEvent(repo *Repository, event, delivery, type_ string, data []byte) error {
	if _, err := core.Index(repo.EventsIndex(), type_, delivery, nil, data); err != nil {
		return err
	}
	return nil
}

func storeGithubSnapshot(repo *Repository, event, delivery, type_ string, data []byte) error {
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
	if _, err := core.Index(repo.SnapshotIndex(), type_, payloadId, nil, payload); err != nil {
		return err
	}
	return nil
}
