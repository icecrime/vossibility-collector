package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
)

type NSQHandler struct {
	Config     *Config
	Repository string
}

func (n *NSQHandler) HandleMessage(m *nsq.Message) error {
	return onMessage(n.Config, n.Repository, m)
}

func main() {
	// Read configuration file.
	config, err := ParseConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}

	// Graceful stop on SIGTERM and SIGINT.
	s := make(chan os.Signal, 64)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)

	// Subscribe to the message queues.
	queues := make([]*Queue, 0, len(config.Repositories))
	for _, conf := range config.Repositories {
		qconf := &NSQConfig{Topic: conf.Topic, Channel: config.NSQ.Channel, Lookupd: config.NSQ.Lookupd}
		queue, err := NewQueue(qconf, &NSQHandler{Config: config, Repository: "docker"})
		if err != nil {
			log.Fatal(err)
		}
		queues = append(queues, queue)
	}

	for {
		select {
		case <-queues[0].Consumer.StopChan:
			log.Debug("Queue stop channel signaled")
			return
		case sig := <-s:
			log.WithField("signal", sig).Debug("received signal")
			queues[0].Consumer.Stop()
		}
	}
}
