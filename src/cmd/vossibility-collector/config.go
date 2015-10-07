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
	ElasticSearch       string
	GithubAPIToken      string
	PeriodicSync        PeriodicSync
	NSQ                 NSQConfig
	NotAnalyzedPatterns []string
	Repositories        map[string]*Repository
}

// configFromFile creates a Config object from its serialized counterpart.
func configFromFile(c *serializedConfig) *Config {
	out := &Config{
		ElasticSearch:       c.ElasticSearch,
		GithubAPIToken:      c.GithubAPIToken,
		NSQ:                 c.NSQ,
		NotAnalyzedPatterns: c.Mapping[MappingNotAnalyzedKey],
		Repositories:        make(map[string]*Repository),
	}

	// Create periodic sync.
	p, err := NewPeriodicSync(c.PeriodicSync)
	if err != nil {
		log.Fatal(err)
	}
	out.PeriodicSync = p

	// Create repositories.
	for name, config := range c.Repositories {
		repo, err := NewRepository(name, &config, c)
		if err != nil {
			log.Fatal(err)
		}
		repo.PeriodicSync = p
		out.Repositories[name] = repo
	}
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
	api.Hosts = append(api.Hosts, config.ElasticSearch)
	return configFromFile(&config), nil
}

// ParseConfigOrDie returns a Config object from the requested filename and
// exits in case of error.
func ParseConfigOrDie(filename string) (c *Config) {
	var err error
	if c, err = ParseConfig(filename); err == nil {
		return c
	}
	log.Fatalf("failed to load configuration file %q: %v", filename, err)
	return nil
}
