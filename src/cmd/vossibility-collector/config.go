package main

import (
	"cmd/vossibility-collector/config"
	"cmd/vossibility-collector/storage"

	log "github.com/Sirupsen/logrus"
	"github.com/mattbaird/elastigo/api"
)

// Config is the global configuration for the tool.
type Config struct {
	ElasticSearch       string
	GitHubAPIToken      string
	PeriodicSync        config.PeriodicSync
	NSQ                 config.NSQConfig
	NotAnalyzedPatterns []string
	Repositories        map[string]*storage.Repository
}

// configFromFile creates a Config object from its serialized counterpart.
func configFromFile(c *config.SerializedConfig) *Config {
	out := &Config{
		ElasticSearch:       c.ElasticSearch,
		GitHubAPIToken:      c.GitHubAPIToken,
		NSQ:                 c.NSQ,
		NotAnalyzedPatterns: c.Mapping[config.MappingNotAnalyzedKey],
		Repositories:        make(map[string]*storage.Repository),
	}

	// Create periodic sync.
	p, err := config.NewPeriodicSync(c.PeriodicSync)
	if err != nil {
		log.Fatal(err)
	}
	out.PeriodicSync = p

	// Create repositories.
	for name, config := range c.Repositories {
		repo, err := storage.NewRepository(name, &config, c)
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
	config, err := config.ParseRawConfiguration(filename)
	if err != nil {
		return nil, err
	}

	// Configure the Elastic Search client library once and for all.
	api.Hosts = append(api.Hosts, config.ElasticSearch)
	return configFromFile(config), nil
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
