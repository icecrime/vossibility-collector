package storage

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mattbaird/elastigo/api"
)

type mockRepoInfo struct{}

func (mockRepoInfo) FullName() string {
	return "mockRepoInfo.FullName"
}

func (mockRepoInfo) PrettyName() string {
	return "mockRepoInfo.PrettyName"
}

func TestFnContext(t *testing.T) {
	c := Context{
		Repository: mockRepoInfo{},
	}

	f := fnContext(c)
	if v := f().Repository.FullName(); v != (mockRepoInfo{}.FullName()) {
		t.Fatalf("unexpected FullName() %q", v)
	}
	if v := f().Repository.PrettyName(); v != (mockRepoInfo{}.PrettyName()) {
		t.Fatalf("unexpected PrettyName() %q", v)
	}
}

func TestFnDaysDifference(t *testing.T) {
	if fnDaysDifference("", "") != nil {
		t.Fatal("expected nil returrn")
	}
}

func TestFnUserData(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users/user/test/_source", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/users/user/icecrime/_source", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(`{
			"company": "Docker",
			"is_maintainer": true
		}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	api.Hosts = []string{srv.URL[7:]}

	if v := fnUserData("test"); *v != (UserData{"test", "", false}) {
		t.Fatalf("unexpected user data for test %v", v)
	}
	if v := fnUserData("icecrime"); *v != (UserData{"icecrime", "Docker", true}) {
		t.Fatalf("unexpected user data for icecrime %v", v)
	}
}

func TestFnUserFunction(t *testing.T) {
	if r, err := fnUserFunction("testdata/test_fn")(); err != nil {
		t.Fatalf("unexpected error result for test_fn %v", err)
	} else if !reflect.DeepEqual(r, map[string]interface{}{
		"key": "value",
	}) {
		t.Fatalf("unexpected result for test_fn %v", r)
	}

	if r, err := fnUserFunction("testdata/missing_fn")(); err == nil {
		t.Fatalf("expected error result for missing function, got %v", r)
	}

	if r, err := fnUserFunction("testdata/test_fn_params")("arg1", "arg2", "arg3"); err != nil {
		t.Fatalf("unexpected error result for test_fn_params %v", err)
	} else if !reflect.DeepEqual(r, map[string]interface{}{
		"key":  "value",
		"args": []interface{}{"arg1", "arg2", "arg3"},
	}) {
		t.Fatalf("unexpected result for test_fn_params %v", r)
	}
}
