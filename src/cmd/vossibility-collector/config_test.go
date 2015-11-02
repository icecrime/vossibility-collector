package main

import "testing"

type testRepo struct{}

func (*testRepo) FullName() string {
	return "user/repo"
}

func (*testRepo) PrettyName() string {
	return "repository (user:repo)"
}

var testContext = Context{
	Repository: &testRepo{},
}

func TestApplyNestedTransformation(t *testing.T) {
	/*
		s := "./testdata/pull_request_event_with_labels.json"
		f, err := ioutil.ReadFile(s)
		if err != nil {
			t.Fatalf("failed to open file %q", s)
		}

		m := map[string]map[string]string{
			"nest": {
				"labels":    "{{ range .labels }}{{ .name }}{{ end }}",
				"merged_by": "{{ if .merged_by }}{{ .merged_by.login }}{{ end }}",
				"number":    "{{ .number }}",
			},
			"pull_request": {
				"action": "{{ .action }}",
				"item":   "{{ apply_transformation \"nest\" .pull_request }}",
			},
		}

		var data map[string]interface{}
		if err := json.Unmarshal(f, &data); err != nil {
			t.Fatal(err)
		}

			tr, err := TransformationsFromConfig(testContext, m)
			if err != nil {
				t.Fatalf("error creating transformation: %v", err)
			}

				r, err := tr["pull_request"].ApplyMap(data)
				if err != nil {
					t.Fatalf("error applying transformation: %v", err)
				}

				json.NewEncoder(os.Stdout).Encode(r)
	*/

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
