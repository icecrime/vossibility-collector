package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/codegangsta/cli"
	"github.com/mattbaird/elastigo/api"
)

var runCommand = cli.Command{
	Name:   "run",
	Usage:  "listen and process Github events",
	Action: doRunCommand,
}

type NSQHandler struct {
	Config     *Config
	Repository *Repository
}

func (n *NSQHandler) HandleMessage(m *nsq.Message) error {
	return onMessage(n.Config, n.Repository, m)
}

func doRunCommand(c *cli.Context) {
	conf := ParseConfigOrDie(c.GlobalString("config"))
	api.Domain = conf.ElasticSearch

	// Graceful stop on SIGTERM and SIGINT.
	s := make(chan os.Signal, 64)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)

	// TODO Make it work with multiple repositories
	// Subscribe to the message queues.
	queues := make([]*Queue, 0, len(conf.Repositories))
	for _, repo := range conf.GetRepositories() {
		qconf := &NSQConfig{Topic: repo.Topic, Channel: conf.NSQ.Channel, Lookupd: conf.NSQ.Lookupd}
		queue, err := NewQueue(qconf, &NSQHandler{Config: conf, Repository: repo})
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
