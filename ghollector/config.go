package main

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bitly/go-simplejson"
)

const (
	// DefaultEventSet is the name of the default set of events to subscribe
	// to. This is the name of set that will be used if it is left unspecified
	// for a given repository.
	//
	// The set doesn't have to exist for a configuration to be valid as long as
	// every repository explicitely refers to a valid event set.
	DefaultEventSet = "default"
)

// Mask is the list of object attributes to persist.
type Mask []string

// Apply takes a serialized JSON payload and returns a JSON payload where only
// the masked fields are preserved, or the original payload if the mask is
// empty.
func (m Mask) Apply(payload []byte) ([]byte, error) {
	if len(m) == 0 {
		return payload, nil
	}

	sj, err := simplejson.NewJson(payload)
	if err != nil {
		return nil, err
	}

	res := simplejson.New()
	for _, e := range m {
		path := strings.Split(e, ".")
		res.SetPath(path, sj.GetPath(path...).Interface())
	}
	return res.Encode()
}

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

// NSQConfig is the configuration for NSQ.
type NSQConfig struct {
	Topic   string `json:"topic"`
	Channel string `json:"channel"`
	Lookupd string `json:"lookup_address"`
}

// RepositoryConfig is the configuration for a given repository.
type RepositoryConfig struct {
	User  string
	Repo  string
	Topic string

	// events is kept internal: use the EventSet() function which properly
	// takes the DefaultEventSet into account.
	events string `toml:"event_set"`
}

// EventSetName returns the name of subscribed events set for the repository.
func (r RepositoryConfig) EventSetName() string {
	if r.events == "" {
		return DefaultEventSet
	}
	return r.events
}

// Config is the global configuration for the tool.
type Config struct {
	NSQ          NSQConfig
	Repositories map[string]RepositoryConfig
	EventSet     map[string]EventSet `toml:"event_set"`
	Masks        map[string]Mask
}

// GetRepositoryEventSet returns the EventSet for the requested repository.
func (c *Config) GetRepositoryEventSet(repo string) (EventSet, error) {
	if r, ok := c.Repositories[repo]; ok {
		eventSetName := r.EventSetName()
		if e, ok := c.EventSet[eventSetName]; ok {
			return e, nil
		}
		return nil, fmt.Errorf("unknown event set %q for repository %q", eventSetName, repo)
	}
	return nil, fmt.Errorf("unknown repository %q", repo)
}

//---------------------------------------------------------------------------//

func ParseConfig(filename string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		return nil, err
	}
	if err := verifyConfig(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func verifyConfig(config *Config) error {
	// We enforce the following rules:
	//
	//   1. Each repository should reference a valid subscription event set
	//   2. Each repository should reference a unique NSQ topic
	//	 3. Each mask must be a valid Github event identifier
	topics := make(map[string]struct{})
	for repo, conf := range config.Repositories {
		// Validate event set
		eventSetName := conf.EventSetName()
		if _, ok := config.EventSet[eventSetName]; !ok {
			return fmt.Errorf("unknown event set %q for repository %q", eventSetName, repo)
		}

		// Validate queue name
		if _, ok := topics[conf.Topic]; ok {
			return fmt.Errorf("duplicated topipc name %q for repository %q", conf.Topic, repo)
		}
		topics[conf.Topic] = struct{}{}
	}
	// Validate masks
	for key, _ := range config.Masks {
		if !isValidEventType(key) {
			return fmt.Errorf("invalid event type %q for mask definition", key)
		}
	}
	return nil
}
