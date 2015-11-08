package github

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gh "github.com/google/go-github/github"
)

func simulateAPIRequest(t *testing.T, c *gh.Client) []*http.Request {
	var requests []*http.Request
	mux := http.NewServeMux()
	mux.HandleFunc("/rate_limit", func(w http.ResponseWriter, req *http.Request) {
		requests = append(requests, req)
		w.Write([]byte("{}"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c.BaseURL, _ = url.Parse(srv.URL)
	if _, _, err := c.RateLimits(); err != nil {
		t.Fatalf("failed to retrieve rate limits; %v", err)
	}
	return requests
}

func TestClientWithoutToken(t *testing.T) {
	c := NewClient("")
	requests := simulateAPIRequest(t, c)
	if len(requests) != 1 {
		t.Fatalf("unexpected number of requests %d, expected 1", len(requests))
	}
	if _, ok := requests[0].Header["Authorization"]; ok {
		t.Fatal("unexpected Authorization header in request")
	}
}

func TestClientWithToken(t *testing.T) {
	c := NewClient("t0k3n")
	requests := simulateAPIRequest(t, c)
	if len(requests) != 1 {
		t.Fatalf("unexpected number of requests %d, expected 1", len(requests))
	}
	if v, ok := requests[0].Header["Authorization"]; !ok || len(v) != 1 {
		t.Fatal("expected single-value Authorization header in request")
	} else if expected := "Bearer t0k3n"; v[0] != expected {
		t.Fatalf("got Authorization token %q, expected %q", v[0], expected)
	}
}
