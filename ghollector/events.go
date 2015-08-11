package main

const (
	EvtCommitComment            = "commit_comment"
	EvtCreate                   = "create"
	EvtDelete                   = "delete"
	EvtDeployment               = "deployment"
	EvtDeploymentStatus         = "deployment_status"
	EvtFollow                   = "follow"
	EvtFork                     = "fork"
	EvtForkApply                = "fork_apply"
	EvtGollum                   = "gollum"
	EvtIssueComment             = "issue_comment"
	EvtIssues                   = "issues"
	EvtMember                   = "member"
	EvtMembership               = "membership"
	EvtPageBuild                = "page_build"
	EvtPublic                   = "public"
	EvtPullRequest              = "pull_request"
	EvtPullRequestReviewComment = "pull_request_review_comment"
	EvtPush                     = "push"
	EvtRelease                  = "release"
	EvtRepositories             = "repositories"
	EvtStatus                   = "status"
	EvtTeamAdd                  = "team_add"
	EvtWatch                    = "watch"
)

var (
	// GithubEventTypes is the set of all possible Github webhooks events.
	GithubEventTypes = map[string]bool{
		EvtCommitComment:            true,
		EvtCreate:                   true,
		EvtDelete:                   true,
		EvtDeployment:               true,
		EvtDeploymentStatus:         true,
		EvtFollow:                   true,
		EvtFork:                     true,
		EvtForkApply:                true,
		EvtGollum:                   true,
		EvtIssueComment:             true,
		EvtIssues:                   true,
		EvtMember:                   true,
		EvtMembership:               true,
		EvtPageBuild:                true,
		EvtPublic:                   true,
		EvtPullRequest:              true,
		EvtPullRequestReviewComment: true,
		EvtPush:                     true,
		EvtRelease:                  true,
		EvtRepositories:             true,
		EvtStatus:                   true,
		EvtTeamAdd:                  true,
		EvtWatch:                    true,
	}

	// GithubSnapshotedEvents is a map of events for which we want to persist
	// the latest version as a snapshot, associated with the identifier of the
	// payload in the event message (yes, it can be different).
	GithubSnapshotedEvents = map[string]string{
		EvtIssues + "_event":      "issue",
		EvtPullRequest + "_event": "pull_request",
	}
)

func IsValidEventType(event string) bool {
	return GithubEventTypes[event]
}
