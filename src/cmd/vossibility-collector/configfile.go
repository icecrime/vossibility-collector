package main

import "fmt"

const (
	// MappingNotAnalyzedKey is the key for the patterns to exclude from
	// Elastic Search analysis.
	MappingNotAnalyzedKey = "not_analyzed"
)

type serializedTable map[string]map[string]string

// serializedConfig is the serialize version of the configuration.
type serializedConfig struct {
	ElasticSearch   string
	GitHubAPIToken  string `toml:"github_api_token"`
	PeriodicSync    string `toml:"sync_periodicity"`
	NSQ             NSQConfig
	Mapping         map[string][]string
	Repositories    map[string]RepositoryConfig
	EventSet        serializedTable `toml:"event_set"`
	Transformations serializedTable
}

// verify enforces several rules about the configuration.
func (c *serializedConfig) verify() error {
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

func (c *serializedConfig) verifyEventSet() error {
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

func (c *serializedConfig) verifyRepositories() error {
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

func (c *serializedConfig) verifyTransformations() error {
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
