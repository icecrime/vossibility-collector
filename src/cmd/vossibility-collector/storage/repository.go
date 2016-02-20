package storage

import (
	"fmt"
	"time"

	"cmd/vossibility-collector/config"
	"cmd/vossibility-collector/transformation"
)

// Repository represents a GitHub repository with its associated user given
// name and subscribed events set.
//
// Data for a repository is stored in Elastic Search according to the following
// structure:
//	- Events in a per-month 'givenName_user-repo_YYYY-MM' index
//	- Current state in a single 'givenname_user-repo_snapshot' index
type Repository struct {
	// RepositoryConfig is the configuration defined for this particular
	// repository.
	config.RepositoryConfig

	// GivenName is the symbolic name for the repository as defined in the
	// configuration.
	GivenName string

	// EventSet is a map of subscribed GitHub events associated with the data
	// transformation to be applied.
	EventSet EventSet

	// PeriodicSync is the synchronization periodicity for this particular
	// repository.
	PeriodicSync config.PeriodicSync

	// Transformations is the collection of transformations instantiated for
	// this particular repository.
	//
	// It is likely that most of them are not repository specific (for example
	// if they never rely on the context information). However, this is a
	// difficult thing to anticipate, and is made even more complicated by the
	// fact that a given transformation can call another one through the
	// provided "apply_transformation" function.
	Transformations *transformation.Transformations
}

func NewRepository(givenName string, repoConfig *config.RepositoryConfig, fullConfig *config.SerializedConfig) (*Repository, error) {
	r := &Repository{
		EventSet:         make(map[string]transformation.Transformation),
		GivenName:        givenName,
		RepositoryConfig: *repoConfig,
	}

	// Initialize repository specific transformations.
	transfo := transformation.NewTransformations()
	initializeTemplateFunc(r, transfo, fullConfig)

	// Load transformations definitions from config.
	if err := transfo.Load(fullConfig.Transformations); err != nil {
		return nil, err
	}
	r.Transformations = transfo

	// Extract the specified event set for this repository.
	evtSetName := repoConfig.EventSetName()
	evtSet, ok := fullConfig.EventSet[evtSetName]
	if !ok {
		return nil, fmt.Errorf("invalid event set %q for repository %q", evtSetName, givenName)
	}

	// Map the transformation to each of the configured events.
	for event, transfoName := range evtSet {
		r.EventSet[event] = r.Transformations.Get(transfoName)
	}
	return r, nil
}

func initializeTemplateFunc(r *Repository, t *transformation.Transformations, fullConfig *config.SerializedConfig) {
	// Register builtin functions.
	context := struct{ Repository RepositoryInfo }{Repository: r}
	t.Funcs["context"] = fnContext(context)
	t.Funcs["days_difference"] = fnDaysDifference
	t.Funcs["user_data"] = fnUserData

	// Register user-defined functions. We do this after the builtins to give an
	// opportunity to override them.
	for name, binary := range fullConfig.Functions {
		t.Funcs[name] = fnUserFunction(binary)
	}
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
	if r.PeriodicSync == config.SyncDaily || r.PeriodicSync == config.SyncWeekly {
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

// IsSubscribed returns whether we should subscribe for a particular GitHub
// event type for this repository.
func (r *Repository) IsSubscribed(event string) bool {
	return r.EventSet.Contains(event)
}

// FullName returns a GitHub compatible identifier for the repository.
func (r *Repository) FullName() string {
	return r.User + "/" + r.Repo
}

// PrettyName returns a human readable identifier for the repository.
func (r *Repository) PrettyName() string {
	return fmt.Sprintf("%s (%s:%s)", r.GivenName, r.User, r.Repo)
}
