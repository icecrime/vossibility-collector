package main

import (
	"cmd/vossibility-collector/config"

	"github.com/bitly/go-nsq"
)

type Queue struct {
	Consumer *nsq.Consumer
}

func NewQueue(config *config.NSQConfig, handler nsq.Handler) (*Queue, error) {
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
