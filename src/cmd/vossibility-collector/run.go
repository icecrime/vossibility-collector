package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"cmd/vossibility-collector/config"
	"cmd/vossibility-collector/github"
	"cmd/vossibility-collector/storage"

	"github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/codegangsta/cli"
	gh "github.com/google/go-github/github"
)

var runCommand = cli.Command{
	Name:   "run",
	Usage:  "listen and process GitHub events",
	Action: doRunCommand,
}

func doRunCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	client := github.NewClient(config.GitHubAPIToken)

	// Create and start monitoring queues.
	lock := sync.RWMutex{}
	queues := createQueues(client, config, &lock)
	stopChan := monitorQueues(queues)

	// Graceful stop on SIGTERM and SIGINT.
	s := make(chan os.Signal, 64)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)

	// Compute next tick time for the synchronization event.
	nextTickTime := resetNextTickTime(config.PeriodicSync)
	for {
		select {
		case <-stopChan:
			logrus.Debug("All queues exited")
			return
		case sig := <-s:
			logrus.WithField("signal", sig).Debug("received signal")
			for _, q := range queues {
				q.Consumer.Stop()
			}
		case <-time.After(nextTickTime):
			lock.Lock() // Take a write lock, which pauses all queue processing.
			logrus.Infof("Starting periodic sync")
			runPeriodicSync(client, config)
			nextTickTime = resetNextTickTime(config.PeriodicSync)
			lock.Unlock()
		}
	}
}

type Queue struct {
	Consumer *nsq.Consumer
}

func NewQueue(config *config.NSQConfig, handler nsq.Handler) (*Queue, error) {
	logger := log.New(os.Stderr, "", log.Flags())
	consumer, err := nsq.NewConsumer(config.Topic, config.Channel, nsq.NewConfig())
	if err != nil {
		return nil, err
	}

	consumer.AddHandler(handler)
	consumer.SetLogger(logger, nsq.LogLevelWarning)
	if err := consumer.ConnectToNSQLookupd(config.Lookupd); err != nil {
		return nil, err
	}

	return &Queue{Consumer: consumer}, nil
}

func createQueues(client *gh.Client, c *Config, lock *sync.RWMutex) []*Queue {
	// Subscribe to the message queues for each repository.
	queues := make([]*Queue, 0, len(c.Repositories))
	for _, repo := range c.Repositories {
		qconf := &config.NSQConfig{
			Topic:   repo.Topic,
			Channel: c.NSQ.Channel,
			Lookupd: c.NSQ.Lookupd,
		}
		queue, err := NewQueue(qconf, NewMessageHandler(client, repo, lock))
		if err != nil {
			logrus.Fatal(err)
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
		go func(queue *Queue) {
			<-queue.Consumer.StopChan
			logrus.Debug("Queue stop channel signaled")
			wg.Done()
		}(q)
	}

	// Multiplex all queues exit into a single channel we can select on.
	stopChan := make(chan struct{})
	go func() {
		wg.Wait()
		stopChan <- struct{}{}
	}()
	return stopChan
}

func resetNextTickTime(p config.PeriodicSync) time.Duration {
	nextTickTime := p.Next()
	logrus.Infof("Next sync in %s (%s)", nextTickTime, time.Now().Add(nextTickTime).Format("Jan 2, 2006 at 15:04:05"))
	return nextTickTime
}

func runPeriodicSync(client *gh.Client, config *Config) {
	// Get the list of repositories.
	repos := make([]*storage.Repository, 0, len(config.Repositories))
	for _, r := range config.Repositories {
		repos = append(repos, r)
	}

	// Run a default synchronization job, with the storage type set to
	// StoreCurrentState (which corresponds to our rolling storage).
	syncOptions := github.DefaultSyncOptions
	syncOptions.SleepPerPage = 10 // TODO Tired of getting blacklisted :-)
	syncOptions.State = github.GitHubStateFilterOpened
	syncOptions.Storage = storage.StoreCurrentState

	// Create the blobStore and run the syncCommand.
	blobStore := storage.NewTransformingBlobStore()
	github.NewSyncCommandWithOptions(client, blobStore, &syncOptions).Run(repos)
}
