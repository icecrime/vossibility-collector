package storage

import (
	"cmd/vossibility-collector/transformation"
)

// EventSet is a map of GitHub event types to subscribe to associated with
// their transformation.
type EventSet map[string]*transformation.Transformation

// Contains returns whether the given eventType belongs in the event set.
func (e EventSet) Contains(eventType string) bool {
	_, ok := e[eventType]
	return ok
}
