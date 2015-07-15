package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/bitly/go-simplejson"
	"github.com/mattbaird/elastigo/core"
)

type MessageHandler struct {
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

	fmt.Printf("%s\n", string(payload))

	// We rely on the fact that a defaut constructed Mask is a pass-through, so
	// we don't need to test if it's even defined.
	data, err := m.Config.Transformations[event].Apply(payload)
	if err != nil {
		log.Errorf("Failed to apply mask for event %q: %v", event, err)
		return err
	}

	// TODO
	// If issue_comment with pull_request attribute, it's a pull_request_comment

	// Store the event in Elastic Search: index is determined by the repository
	// and type by the event type.
	if err := storeGithubEvent(m.Repo, event, delivery, data); err != nil {
		return err
	}
	if err := storeGithubSnapshot(m.Repo, event, delivery, data); err != nil {
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
