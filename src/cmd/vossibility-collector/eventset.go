package main

const (
	// DefaultEventSet is the name of the default set of events to subscribe
	// to. This is the name of set that will be used if it is left unspecified
	// for a given repository.
	//
	// The set doesn't have to exist for a configuration to be valid as long as
	// every repository explicitely refers to a valid event set.
	DefaultEventSet = "default"
)

// EventSet is a map of GitHub event types to subscribe to associated with
// their transformation.
type EventSet map[string]*Transformation

// Contains returns whether the given eventType belongs in the event set.
func (e EventSet) Contains(eventType string) bool {
	_, ok := e[eventType]
	return ok
}
