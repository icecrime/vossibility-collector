package github

import (
	"net/http"

	gh "github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func NewClient(token string) *gh.Client {
	var tc *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		})
		tc = oauth2.NewClient(oauth2.NoContext, ts)
	}
	return gh.NewClient(tc)
}
