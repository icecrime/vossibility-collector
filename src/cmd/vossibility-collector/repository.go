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
	RepositoryConfig
	GivenName    string
	EventSet     EventSet
	PeriodicSync PeriodicSync
}

// periodFormat returns the string representation of a timestamp to be used in
// the time-based indices.
func periodFormat(timestamp time.Time, format string) string {
	// Notices the UTC here: it's a bit counterintuitive for the user (because
	// you end up potentially seeing indices names in the future), but that's
	// how Kibana and ES work.
	// Reference: https://groups.google.com/forum/#!topic/logstash-users/_sdJNWJ4_5g
	return timestamp.UTC().Format(format)
}

// dailyPeriodFormat returns the string representation of a timestamp to be
// used in an daily time-based indices.
func dailyPeriodFormat(timestamp time.Time) string {
	return periodFormat(timestamp, "2006.01.02")
}

// monthlyPeriodFormat returns the string representation of a timestamp to be
// used in an hourly time-based indices.
func monthlyPeriodFormat(timestamp time.Time) string {
	return periodFormat(timestamp, "2006.01")
}

// hourlyPeriodFormat returns the string representation of a timestamp to be
// used in an hourly time-based indices.
func hourlyPeriodFormat(timestamp time.Time) string {
	return periodFormat(timestamp, "2006.01.02-15")
}

// IndexPrefix returns the string that prefixes all Elastic Search indices for
// this repository data.
func (r *Repository) IndexPrefix() string {
	return r.GivenName + "-"
}

// LiveIndex returns the current Elastic Search index appropriate to store this
// repository's events. This value changes over time.
func (r *Repository) LiveIndex() string {
	return r.LiveIndexForTimestamp(time.Now())
}

// LiveIndexForTimestamp returns the current Elastic Search index appropriate
// to store this repository's events with the specified timestamp.
func (r *Repository) LiveIndexForTimestamp(timestamp time.Time) string {
	return fmt.Sprintf("%slive-%s", r.IndexPrefix(), monthlyPeriodFormat(timestamp))
}

// StateIndex returns the current Elastic Search index appropriate to store
// this repository's items current state. This value changes over time.
func (r *Repository) StateIndex() string {
	return r.StateIndexForTimestamp(time.Now())
}

// StateIndexForTimestamp returns the Elastic Search index appropriate to store
// an object with the specified timestamp.
func (r *Repository) StateIndexForTimestamp(timestamp time.Time) string {
	// The state index depends on the chosen sync periodicity.
	format := hourlyPeriodFormat(timestamp)
	if r.PeriodicSync == SyncDaily || r.PeriodicSync == SyncWeekly {
		format = dailyPeriodFormat(timestamp)
	}
	return fmt.Sprintf("%sstate-%s", r.IndexPrefix(), format)
}

// SnapshotIndex returns the current Elastic Search index appropriate to store
// this repository's snapshot data (such as the latest state of each pull
// request and issue).
func (r *Repository) SnapshotIndex() string {
	return fmt.Sprintf("%ssnapshot", r.IndexPrefix())
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
