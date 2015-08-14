package main

import (
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/mattbaird/elastigo/api"
)

// NSQConfig is the configuration for NSQ.
type NSQConfig struct {
	Topic   string `json:"topic"`
	Channel string `json:"channel"`
	Lookupd string `json:"lookup_address"`
}

// RepositoryConfig is the configuration for a given repository.
type RepositoryConfig struct {
	User       string
	Repo       string
	Topic      string
	StartIndex int `toml:"start_index"`

	// events is kept internal: use the EventSetName() function which properly
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

// configFromFile creates a Config object from its serialized counterpart.
func configFromFile(c *serializedConfig) *Config {
	out := &Config{
		ElasticSearch:  c.ElasticSearch,
		GithubApiToken: c.GithubApiToken,
		NSQ:            c.NSQ,
		EventSet:       make(map[string]EventSet),
		Repositories:   make(map[string]*Repository),
	}
	// Create transformations.
	t, err := TransformationsFromConfig(c.Transformations)
	if err != nil {
		log.Fatal(err)
	}
	out.Transformations = t
	// Create event sets.
	for name, d := range c.EventSet {
		set := EventSet(make(map[string]*Transformation))
		for event, transformation := range d {
			set[event] = out.Transformations[transformation]
		}
		out.EventSet[name] = set
	}
	// Create repositories.
	for name, config := range c.Repositories {
		evt := config.EventSetName()
		out.Repositories[name] = &Repository{
			GivenName:        name,
			EventSet:         out.EventSet[evt],
			RepositoryConfig: &config,
		}
	}
	// Initialize periodic sync.
	p, err := NewPeriodicSync(c.PeriodicSync)
	if err != nil {
		log.Fatal(err)
	}
	out.PeriodicSync = p
	return out
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
