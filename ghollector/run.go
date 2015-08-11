package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

var runCommand = cli.Command{
	Name:   "run",
	Usage:  "listen and process Github events",
	Action: doRunCommand,
}

func doRunCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	client := NewClient(config)

	// Create and start monitoring queues.
	queues := createQueues(client, config)
	stopChan := monitorQueues(queues)

	// Graceful stop on SIGTERM and SIGINT.
	s := make(chan os.Signal, 64)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)

	// Compute next tick time for the synchronization event.A
	nextTickTime := resetNextTickTime(config.PeriodicSync)
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
		case <-time.After(nextTickTime):
			runPeriodicSync(client, config)
			nextTickTime = resetNextTickTime(config.PeriodicSync)
		}
	}
}

func createQueues(client *github.Client, config *Config) []*Queue {
	// Subscribe to the message queues for each repository.
	queues := make([]*Queue, 0, len(config.Repositories))
	for _, repo := range config.Repositories {
		qconf := &NSQConfig{
			Topic:   repo.Topic,
			Channel: config.NSQ.Channel,
			Lookupd: config.NSQ.Lookupd,
		}
		queue, err := NewQueue(qconf, NewMessageHandler(client, config, repo))
		if err != nil {
			log.Fatal(err)
		}
		queues = append(queues, queue)
	}
	return queues
}

func monitorQueues(queues []*Queue) <-chan struct{} {
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
	return stopChan
}

func resetNextTickTime(p PeriodicSync) time.Duration {
	nextTickTime := p.Next()
	log.Infof("Next sync in %s (%s)", nextTickTime, time.Now().Add(nextTickTime).Format("Jan 2, 2006 at 15:04:05"))
	return nextTickTime
}

func runPeriodicSync(client *github.Client, config *Config) {
	// Get the list of repositories.
	repos := make([]*Repository, 0, len(config.Repositories))
	for _, r := range config.Repositories {
		repos = append(repos, r)
	}

	// Run a default synchronization job, with the storage type set to
	// StoreCurrentState (which corresponds to our rolling storage).
	syncOptions := DefaultSyncOptions
	syncOptions.SleepPerPage = 10 // TODO Tired of getting blacklisted :-)
	syncOptions.Storage = StoreCurrentState

	// Create the blobStore and run the syncCommand.
	blobStore := NewTransformingBlobStore(config.Transformations)
	NewSyncCommandWithOptions(client, blobStore, &syncOptions).Run(repos)
}
