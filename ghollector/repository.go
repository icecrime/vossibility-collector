package main

import (
	"fmt"
	"time"
)

// Repository represents a Github repository with its associated user given
// name and subscribed events set.
//
// Data for a repository is stored in Elastic Search according to the following
// structure:
//	- Events in a per-month 'givenName_user-repo_YYYY-MM' index
//	- Current state in a single 'givenname_user-repo_snapshot' index
type Repository struct {
	*RepositoryConfig
	GivenName string
	EventSet  EventSet
}

func (r *Repository) indexPrefix() string {
	return fmt.Sprintf("%s_%s-%s", r.GivenName, r.User, r.Repo)
}

// EventsIndex returns the current Elastic Search index appropriate to store
// this repository's events. This value changes over time.
func (r *Repository) EventsIndex() string {
	period := time.Now().Format("2006-01")
	return fmt.Sprintf("%s_%s", r.indexPrefix(), period)
}

// SnapshotIndex returns the current Elastic Search index appropriate to store
// this repository's snapshot data (such as the latest state of each pull
// request and issue).
func (r *Repository) SnapshotIndex() string {
	return fmt.Sprintf("%s_snapshot", r.indexPrefix())
}

// IsSubscribed returns whether we should subscribe for a particular Github
// event type for this repository.
func (r *Repository) IsSubscribed(event string) bool {
	return r.EventSet.Contains(event)
}

// PrettyName returns a human readable identifier for the repository.
func (r *Repository) PrettyName() string {
	return fmt.Sprintf("%s (%s:%s)", r.GivenName, r.User, r.Repo)
}
