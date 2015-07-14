package main

var (
	// GithubEventTypes is the set of all possible Github webhooks events.
	GithubEventTypes = map[string]bool{
		"pull_request_review_comment": true,
		"commit_comment":              true,
		"create":                      true,
		"delete":                      true,
		"deployment":                  true,
		"deployment_status":           true,
		"follow":                      true,
		"fork":                        true,
		"fork_apply":                  true,
		"gollum":                      true,
		"issue_comment":               true,
		"issues":                      true,
		"member":                      true,
		"membership":                  true,
		"page_build":                  true,
		"public":                      true,
		"pull_request":                true,
		"push":                        true,
		"release":                     true,
		"repositories":                true,
		"status":                      true,
		"team_add":                    true,
		"watch":                       true,
	}

	// GithubSnapshotedEvents is a map of events for which we want to persist
	// the latest version as a snapshot, associated with the identifier of the
	// payload in the event message.
	GithubSnapshotedEvents = map[string]string{
		"issues":       "issue",
		"pull_request": "pull_request",
	}
)

func IsValidEventType(event string) bool {
	return GithubEventTypes[event]
}
