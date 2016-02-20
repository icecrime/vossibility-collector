package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

const (
	// MappingNotAnalyzedKey is the key for the patterns to exclude from
	// Elastic Search analysis.
	MappingNotAnalyzedKey = "not_analyzed"

	// DefaultEventSet is the name of the default set of events to subscribe
	// to. This is the name of set that will be used if it is left unspecified
	// for a given repository.
	//
	// The set doesn't have to exist for a configuration to be valid as long as
	// every repository explicitely refers to a valid event set.
	DefaultEventSet = "default"
)

const (
	GitHubTypeIssue         = "issue"
	GitHubTypePullRequest   = "pull_request"
	SnapshotIssueType       = "snapshot_issue"
	SnapshotPullRequestType = "snapshot_pull_request"
)

const (
	// MetadataType is the key for the type metadata attribute used when
	// storing a Blob instance into the Elastic Search backend.
	MetadataType = "_type"

	// MetadataSnapshotID is the key for the snapshot id metadata attribute
	// used when storing a Blob instance into the Elastic Search backend. It
	// represents the Id to be used when storing the snapshoted content of a
	// blob.
	MetadataSnapshotID = "_snapshot_id"

	// MetadataSnapshotField is the key for the snapshot id metadata attribute
	// used when storing a Blob instance into the Elastic Search backend. It
	// represents the nested object to be used when storing the snapshoted
	// content of a blob.
	MetadataSnapshotField = "_snapshot_field"
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

type SerializedTable map[string]map[string]string

// SerializedConfig is the serialize version of the configuration.
type SerializedConfig struct {
	ElasticSearch   string
	GitHubAPIToken  string `toml:"github_api_token"`
	PeriodicSync    string `toml:"sync_periodicity"`
	NSQ             NSQConfig
	Functions       map[string]string
	Mapping         map[string][]string
	Repositories    map[string]RepositoryConfig
	EventSet        SerializedTable `toml:"event_set"`
	Transformations SerializedTable
}

func ParseRawConfiguration(filename string) (*SerializedConfig, error) {
	var config SerializedConfig
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		return nil, err
	}
	if err := config.verify(); err != nil {
		return nil, err
	}
	return &config, nil
}

// verify enforces several rules about the configuration.
func (c *SerializedConfig) verify() error {
	for _, fn := range []func() error{
		c.verifyEventSet,
		c.verifyRepositories,
		c.verifyTransformations,
	} {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func (c *SerializedConfig) verifyEventSet() error {
	// Each event in an event set should reference a valid transformation.
	for name, s := range c.EventSet {
		for _, transfo := range s {
			if _, ok := c.Transformations[transfo]; !ok {
				return fmt.Errorf("event %q references an unknown transformation %q", name, transfo)
			}
		}
		// Some events are mandatory.
		for _, mandatoryEvents := range []string{SnapshotIssueType, SnapshotPullRequestType} {
			if _, ok := s[mandatoryEvents]; !ok {
				return fmt.Errorf("missing required event %q in event_set %q", mandatoryEvents, name)
			}
		}
	}
	return nil
}

func (c *SerializedConfig) verifyRepositories() error {
	topics := make(map[string]struct{})
	for repo, conf := range c.Repositories {
		// Validate event set.
		eventSetName := conf.EventSetName()
		if _, ok := c.EventSet[eventSetName]; !ok {
			return fmt.Errorf("unknown event set %q for repository %q", eventSetName, repo)
		}
		// Validate queue name.
		if _, ok := topics[conf.Topic]; ok {
			return fmt.Errorf("duplicated topic name %q for repository %q", conf.Topic, repo)
		}
		topics[conf.Topic] = struct{}{}
	}
	return nil
}

func (c *SerializedConfig) verifyTransformations() error {
	// Transformations should have either none or both of the snapshot
	// metadata fields.
	for name, t := range c.Transformations {
		_, hasSnapshotID := t[MetadataSnapshotID]
		_, hasSnapshotField := t[MetadataSnapshotField]
		if hasSnapshotID != hasSnapshotField {
			return fmt.Errorf("transformation %q should have either none of both attributes %q and %q", name, MetadataSnapshotID, MetadataSnapshotField)
		}
	}
	return nil
}
