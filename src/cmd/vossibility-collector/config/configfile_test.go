package config

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestConfigVerifyEventSet(t *testing.T) {
	c1 := `
[event_set.default]
test1 = "transfo1"
test2 = "transfo2"

[transformations.transfo1]
`

	var config1 SerializedConfig
	if _, err := toml.Decode(c1, &config1); err != nil {
		t.Fatalf("error parsing configuration: %v", err)
	}

	err := config1.verify()
	if expected := "references an unknown transformation"; err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q error, got %v", expected, err)
	}

	c2 := `
[event_set.default]
test1 = "transfo1"
test2 = "transfo2"
snapshot_issue = "test"

[transformations.test]
[transformations.transfo1]
[transformations.transfo2]
`

	var config2 SerializedConfig
	if _, err := toml.Decode(c2, &config2); err != nil {
		t.Fatalf("error parsing configuration: %v", err)
	}

	err = config2.verify()
	if expected := "missing required event"; err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q error, got %v", expected, err)
	}
}
