package main

import (
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/bitly/go-simplejson"
)

func TestApplyMask(t *testing.T) {
	s := "./testdata/pull_request_event.json"
	f, err := ioutil.ReadFile(s)
	if err != nil {
		t.Fatalf("failed to open file %q", s)
	}

	m := Mask{
		"action",
		"number",
		"pull_request.number",
		"pull_request.state",
		"pull_request.title",
		"pull_request.body",
		"pull_request.user.login",
		"pull_request.locked",
		"pull_request.created_at",
		"pull_request.updated_at",
		"pull_request.closed_at",
		"pull_request.merged_at",
		"pull_request.merged",
		"pull_request.mergeable",
		"pull_request.merge_by.login",
		"pull_request.comments",
		"pull_request.commits",
		"pull_request.additions",
		"pull_request.deletions",
		"pull_request.changed_files",
	}
	r, err := m.Apply(f)
	if err != nil {
		t.Fatalf("error applying mask: %v", err)
	}

	res, err := simplejson.NewJson(r)
	if err != nil {
		t.Fatalf("error unserializing masked result: %v", err)
	}

	input, _ := simplejson.NewJson(f)
	for _, v := range m {
		var (
			ok  bool
			tmp *simplejson.Json = res
		)
		path := strings.Split(v, ".")
		for _, p := range path {
			if tmp, ok = tmp.CheckGet(p); !ok {
				t.Fatalf("missing field %q in masked result", v)
			}
		}
		if !reflect.DeepEqual(input.GetPath(path...).Interface(), res.GetPath(path...).Interface()) {
			t.Fatalf("input and result have different value for field %q", path)
		}
	}
}
