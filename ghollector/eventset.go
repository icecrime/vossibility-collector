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

// EventSet is a list of Github event types to subscribe to.
type EventSet []string

// Contains returns whether the given eventType belongs in the event set.
func (e EventSet) Contains(eventType string) bool {
	for _, v := range e {
		if v == eventType {
			return true
		}
	}
	return false
}
