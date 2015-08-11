package main

import "fmt"

// serializedConfig is the serialize version of the configuration.
type serializedConfig struct {
	ElasticSearch   string
	GithubApiToken  string `toml:"github_api_token"`
	PeriodicSync    string `toml:"sync_periodicity"`
	NSQ             NSQConfig
	Repositories    map[string]RepositoryConfig
	EventSet        map[string]EventSet `toml:"event_set"`
	Transformations map[string]map[string]string
}

// getRepository returns the Repository associated with the provided givenName.
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

// getRepositoryEventSet returns the subscribed event set for a particular
// repository, or an error if the event set for that repository is not found.
func (c *serializedConfig) getRepositoryEventSet(r *RepositoryConfig) (EventSet, error) {
	eventSetName := r.EventSetName()
	if e, ok := c.EventSet[eventSetName]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("unknown event set %q for repository %s:%s", eventSetName, r.User, r.Repo)
}

// verify enforces several rules about the configuration:
//  - Each repository should reference a valid subscription event set
//  - Each repository should reference a unique NSQ topic
//	- Each transformation should define either none or both metadata snapshot
//    fields
func (c *serializedConfig) verify() error {
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
