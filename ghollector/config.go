package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/mattbaird/elastigo/api"
)

const (
	// DefaultEventSet is the name of the default set of events to subscribe
	// to. This is the name of set that will be used if it is left unspecified
	// for a given repository.
	//
	// The set doesn't have to exist for a configuration to be valid as long as
	// every repository explicitely refers to a valid event set.
	DefaultEventSet = "default"
)

// EventSet is a list of Github event types to subscribe to.
type EventSet []string

// Contains returns whether the given eventType belongs in the event set.
func (e EventSet) Contains(eventType string) bool {
	for _, v := range e {
		if v == eventType {
			return true
		}
	}
	return false
}

// NSQConfig is the configuration for NSQ.
type NSQConfig struct {
	Topic   string `json:"topic"`
	Channel string `json:"channel"`
	Lookupd string `json:"lookup_address"`
}

// RepositoryConfig is the configuration for a given repository.
type RepositoryConfig struct {
	User  string
	Repo  string
	Topic string

	// events is kept internal: use the EventSet() function which properly
	// takes the DefaultEventSet into account.
	events string `toml:"event_set"`
}

// EventSetName returns the name of subscribed events set for the repository.
func (r RepositoryConfig) EventSetName() string {
	if r.events == "" {
		return DefaultEventSet
	}
	return r.events
}

type serializedConfig struct {
	ElasticSearch   string
	GithubApiToken  string `toml:"github_api_token"`
	PeriodicSync    string `toml:"sync_periodicity"`
	NSQ             NSQConfig
	Repositories    map[string]RepositoryConfig
	EventSet        map[string]EventSet `toml:"event_set"`
	Transformations map[string]map[string]string
}

// Config is the global configuration for the tool.
type Config struct {
	ElasticSearch   string
	GithubApiToken  string
	PeriodicSync    PeriodicSync
	NSQ             NSQConfig
	Repositories    map[string]*Repository
	EventSet        map[string]EventSet
	Transformations Transformations
}

func configFromFile(c *serializedConfig) *Config {
	out := &Config{
		ElasticSearch:   c.ElasticSearch,
		GithubApiToken:  c.GithubApiToken,
		NSQ:             c.NSQ,
		EventSet:        c.EventSet,
		Repositories:    make(map[string]*Repository),
		Transformations: make(map[string]*Transformation),
	}
	for name, _ := range c.Repositories {
		r, err := c.getRepository(name)
		if err != nil {
			log.Fatalf("corrupted repository %q in configuration: %v", name, err)
		}
		out.Repositories[name] = r
	}

	t, err := TransformationsFromConfig(c.Transformations)
	if err != nil {
		log.Fatal(err)
	}
	out.Transformations = t

	p, err := NewPeriodicSync(c.PeriodicSync)
	if err != nil {
		log.Fatal(err)
	}
	out.PeriodicSync = p
	return out
}

// GetRepositories returns the list of all known repositories. It assumes a
// valid configuration and exits if it fails to build the result.
func (c *serializedConfig) getRepositories() []*Repository {
	repos := make([]*Repository, 0, len(c.Repositories))
	for givenName, _ := range c.Repositories {
		r, err := c.getRepository(givenName)
		if err != nil {
			log.Fatalf("corrupted configuration: %v", err)
		}
		repos = append(repos, r)
	}
	return repos
}

// GetRepository returns the Repository associated with the provided givenName.
func (c *serializedConfig) getRepository(givenName string) (*Repository, error) {
	repoConfig, ok := c.Repositories[givenName]
	if !ok {
		return nil, fmt.Errorf("unknown repository %q", givenName)
	}
	eventSet, err := c.getRepositoryEventSet(&repoConfig)
	if err != nil {
		return nil, err
	}
	return &Repository{
		GivenName:        givenName,
		EventSet:         eventSet,
		RepositoryConfig: &repoConfig,
	}, nil
}

func (c *serializedConfig) getRepositoryEventSet(r *RepositoryConfig) (EventSet, error) {
	eventSetName := r.EventSetName()
	if e, ok := c.EventSet[eventSetName]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("unknown event set %q for repository %s:%s", eventSetName, r.User, r.Repo)
}

func (c *serializedConfig) verify() error {
	// We enforce the following rules:
	//   1. Each repository should reference a valid subscription event set
	//   2. Each repository should reference a unique NSQ topic
	//	 3. Each transformation must be a valid Github event identifier
	topics := make(map[string]struct{})
	for repo, conf := range c.Repositories {
		// Validate event set
		eventSetName := conf.EventSetName()
		if _, ok := c.EventSet[eventSetName]; !ok {
			return fmt.Errorf("unknown event set %q for repository %q", eventSetName, repo)
		}

		// Validate queue name
		if _, ok := topics[conf.Topic]; ok {
			return fmt.Errorf("duplicated topipc name %q for repository %q", conf.Topic, repo)
		}
		topics[conf.Topic] = struct{}{}
	}
	return nil
}

// ParseConfig returns a Config object from the requested filename and any
// error encountered during load.
func ParseConfig(filename string) (*Config, error) {
	var config serializedConfig
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		return nil, err
	}
	if err := config.verify(); err != nil {
		return nil, err
	}

	// Configure the Elastic Search client library once and for all.
	api.Domain = config.ElasticSearch
	return configFromFile(&config), nil
}

// ParseConfigOrDie returns a Config object from the requested filename and
// exits in case of error.
func ParseConfigOrDie(filename string) *Config {
	if c, err := ParseConfig(filename); err == nil {
		return c
	} else {
		log.Fatalf("failed to load configuration file %q: %v", filename, err)
	}
	return nil
}
