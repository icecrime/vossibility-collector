package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var runCommand = cli.Command{
	Name:   "run",
	Usage:  "listen and process Github events",
	Action: doRunCommand,
}

func doRunCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))

	// Graceful stop on SIGTERM and SIGINT.
	s := make(chan os.Signal, 64)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)

	// Subscribe to the message queues for each repository.
	queues := make([]*Queue, 0, len(config.Repositories))
	for _, repo := range config.GetRepositories() {
		qconf := &NSQConfig{
			Topic:   repo.Topic,
			Channel: config.NSQ.Channel,
			Lookupd: config.NSQ.Lookupd,
		}
		queue, err := NewQueue(qconf, &MessageHandler{
			Config: config,
			Repo:   repo,
		})
		if err != nil {
			log.Fatal(err)
		}
		queues = append(queues, queue)
	}

	// Start one goroutine per queue and monitor the StopChan event.
	wg := sync.WaitGroup{}
	for _, q := range queues {
		wg.Add(1)
		go func() {
			<-q.Consumer.StopChan
			log.Debug("Queue stop channel signaled")
			wg.Done()
		}()
	}

	// Multiplex all queues exit into a single channel we can select on.
	stopChan := make(chan struct{})
	go func() {
		wg.Wait()
		stopChan <- struct{}{}
	}()

	for {
		select {
		case <-stopChan:
			log.Debug("All queues exited")
			return
		case sig := <-s:
			log.WithField("signal", sig).Debug("received signal")
			for _, q := range queues {
				q.Consumer.Stop()
			}
		}
	}
}
