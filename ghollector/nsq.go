package main

import (
	"encoding/json"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
)

type Queue struct {
	Consumer *nsq.Consumer
}

func NewQueue(config *NSQConfig, handler nsq.Handler) (*Queue, error) {
	consumer, err := nsq.NewConsumer(config.Topic, config.Channel, nsq.NewConfig())
	if err != nil {
		return nil, err
	}

	consumer.AddHandler(handler)
	if err := consumer.ConnectToNSQLookupd(config.Lookupd); err != nil {
		return nil, err
	}

	return &Queue{Consumer: consumer}, nil
}

type NSQCallback func(event, delivery string, payload json.RawMessage) error

type NSQHandler struct {
	Callback NSQCallback
}

func (n *NSQHandler) HandleMessage(m *nsq.Message) error {
	var p partialMessage
	if err := json.Unmarshal(m.Body, &p); err != nil {
		log.Error(err)
		return nil // No need to retry
	}
	return n.Callback(p.GithubEvent, p.GithubDelivery, m.Body)
}
