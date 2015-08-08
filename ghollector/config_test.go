package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/icecrime/vossibility/ghollector/template"
)

func TestApplyTransformation(t *testing.T) {
	s := "./testdata/pull_request_event.json"
	f, err := ioutil.ReadFile(s)
	if err != nil {
		t.Fatalf("failed to open file %q", s)
	}

	m := map[string]string{
		"merged_by": "{{ if .pull_request.merged_by }}{{ .pull_request.merged_by.login }}{{ end }}",
	}

	tr, err := TransformationFromConfig("pull_request", m, template.FuncMap{})
	if err != nil {
		t.Fatalf("error creating transformation: %v", err)
	}

	r, err := tr.Apply(f)
	if err != nil {
		t.Fatalf("error applying transformation: %v", err)
	}

	fmt.Printf("%#v\n", r)
	if m, err := r.Data.Map(); err == nil {
		fmt.Printf("%#v\n", m)
	}

	/*
		res, err := simplejson.NewJson(r.Data)
		if err != nil {
			t.Fatalf("error unserializing transformed result: %v", err)
		}


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
	*/
}

func TestApplyNestedTransformation(t *testing.T) {
	s := "./testdata/pull_request_event.json"
	f, err := ioutil.ReadFile(s)
	if err != nil {
		t.Fatalf("failed to open file %q", s)
	}

	m := map[string]map[string]string{
		"nest": {
			"merged_by": "{{ if .merged_by }}{{ .merged_by.login }}{{ end }}",
			"number":    "",
		},
		"pull_request": {
			"action": "",
			"item":   "{{ apply_transformation \"nest\" .pull_request }}",
		},
	}

	tr, err := TransformationsFromConfig(m)
	if err != nil {
		t.Fatalf("error creating transformation: %v", err)
	}

	r, err := tr["pull_request"].Apply(f)
	if err != nil {
		t.Fatalf("error applying transformation: %v", err)
	}

	fmt.Printf("%#v\n", r)
	if m, err := r.Data.Map(); err == nil {
		fmt.Printf("%#v\n", m)
	}

	/*
		res, err := simplejson.NewJson(r.Data)
		if err != nil {
			t.Fatalf("error unserializing transformed result: %v", err)
		}


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
	*/
}
