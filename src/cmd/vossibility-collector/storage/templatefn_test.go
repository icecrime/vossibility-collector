package storage

import (
	"net/http"
	"net/http/httptest"
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
