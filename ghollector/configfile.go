package main

import "fmt"

// serializedConfig is the serialize version of the configuration.
type serializedConfig struct {
	ElasticSearch   string
	GithubApiToken  string `toml:"github_api_token"`
	PeriodicSync    string `toml:"sync_periodicity"`
	NSQ             NSQConfig
	Repositories    map[string]RepositoryConfig
	EventSet        map[string]map[string]string `toml:"event_set"`
	Transformations map[string]map[string]string
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
	// Some transformations are mandatory.
	for _, mandatoryTransfo := range []string{EvtIssues, EvtPullRequest} {
		if _, ok := c.Transformations[mandatoryTransfo]; !ok {
			return fmt.Errorf("missing required transformation %q", mandatoryTransfo)
		}
	}
	// Transformations should have either none or both of the snapshot
	// metadata fields.
	for name, t := range c.Transformations {
		_, hasSnapshotId := t[MetadataSnapshotId]
		_, hasSnapshotField := t[MetadataSnapshotField]
		if hasSnapshotId != hasSnapshotField {
			return fmt.Errorf("transformation %q should have either none of both attributes %q and %q", name, MetadataSnapshotId, MetadataSnapshotField)
		}
	}
	return nil
}
