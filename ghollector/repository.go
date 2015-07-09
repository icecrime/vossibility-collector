package main

import "fmt"

type Repository struct {
	*RepositoryConfig
	GivenName string
	EventSet  EventSet
}

func (r *Repository) EventsIndex() string {
	return fmt.Sprintf("events-%s-%s", r.User, r.Repo)
}

func (r *Repository) LatestIndex() string {
	return fmt.Sprintf("latest-%s-%s", r.User, r.Repo)
}

func (r *Repository) IsSubscribed(event string) bool {
	return r.EventSet.Contains(event)
}

func (r *Repository) PrettyName() string {
	return fmt.Sprintf("%s (%s:%s)", r.GivenName, r.User, r.Repo)
}
