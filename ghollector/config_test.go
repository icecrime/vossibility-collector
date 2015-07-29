package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/bitly/go-simplejson"
)

func TestApplyTransformation(t *testing.T) {
	s := "./testdata/pull_request_event.json"
	f, err := ioutil.ReadFile(s)
	if err != nil {
		t.Fatalf("failed to open file %q", s)
	}

	m := map[string]string{
		"action":        "",
		"additions":     "{{ .pull_request.additions }}",
		"body":          "{{ .pull_request.body }}",
		"changed_files": "{{ .pull_request.changed_files }}",
		"closed_at":     "{{ .pull_request.closed_at }}",
		"comments":      "{{ .pull_request.comments }}",
		"commits":       "{{ .pull_request.commits }}",
		"created_at":    "{{ .pull_request.created_at }}",
		"deletions":     "{{ .pull_request.deletions }}",
		"locked":        "{{ .pull_request.locked }}",
		"mergeable":     "{{ .pull_request.mergeable }}",
		"merged":        "{{ .pull_request.merged }}",
		"merged_at":     "{{ .pull_request.merged_at }}",
		"merged_by":     "{{ if .pull_request.merged_by }}{{ .pull_request.merged_by.login }}{{ end }}",
		"number":        "{{ .pull_request.number }}",
		"state":         "{{ .pull_request.state }}",
		"title":         "{{ .pull_request.title }}",
		"updated_at":    "{{ .pull_request.updated_at }}",
		"user":          "{{ .pull_request.user.login }}",
	}

	tr, err := FromConfig(m)
	if err != nil {
		t.Fatalf("error creating transformation: %v", err)
	}

	r, err := tr.Apply(f)
	if err != nil {
		t.Fatalf("error applying transformation: %v", err)
	}

	res, err := simplejson.NewJson(r)
	if err != nil {
		t.Fatalf("error unserializing transformed result: %v", err)
	}

	fmt.Printf("%s\n", string(r))

	for k, _ := range m {
		var (
			ok  bool
			tmp *simplejson.Json = res
		)
		path := strings.Split(k, ".")
		for _, p := range path {
			if tmp, ok = tmp.CheckGet(p); !ok {
				t.Fatalf("missing field %q in masked result", k)
			}
		}
	}
}
