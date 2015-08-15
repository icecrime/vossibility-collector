package main

import (
	"net/http"

	gh "github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func NewClient(config *Config) *gh.Client {
	var tc *http.Client
	if config.GithubAPIToken != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: config.GithubAPIToken,
		})
		tc = oauth2.NewClient(oauth2.NoContext, ts)
	}
	return gh.NewClient(tc)
}
