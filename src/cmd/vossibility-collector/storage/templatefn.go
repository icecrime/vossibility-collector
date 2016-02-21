package storage

import (
	"encoding/json"
	"os"
	"os/exec"
	"time"

	"cmd/vossibility-collector/config"
)

// Context is provided to transformations as a way to pass additional data to
// existing templates.
type Context struct {
	// Repository gives information about the repository concerned by the event
	// under transformation.
	Repository RepositoryInfo
}

// RepositoryInfo provides information about the repository to the executed
// templates.
type RepositoryInfo interface {
	// FullName returns the GitHub repository full name, whic is in the form
	// "user/repo" (e.g., "icecrime/docker")
	FullName() string

	// PrettyName returns a vossibility specific string which identifies the
	// repository but also includes its given name (which has no existence on
	// GitHub).
	PrettyName() string
}

// fnContext is a constant function passed down to templates as a mean to
// access contextual information on the data being transformed (e.g.,
// repository information).
func fnContext(context Context) func() Context {
	return func() Context {
		return context
	}
}

// fnDaysDifference computes the difference between two GitHub formatted dates
// and returns it as a floating number of days.
//
// This is not general purpose enough, but does the job for most of the metrics
// that matter to us today (e.g., number of days to close a pull request).
func fnDaysDifference(lhs, rhs string) interface{} {
	lhsT, err := time.Parse(time.RFC3339, lhs)
	if err != nil {
		return nil
	}
	rhsT, err := time.Parse(time.RFC3339, rhs)
	if err != nil {
		return nil
	}
	return lhsT.Sub(rhsT).Hours() / 24
}

// fnUserData enriches the login information with data in database, such as the
// fact that a user is a maintainer or works for company X. It always returns
// a UserData instance, which will only contain the login information when we
// don't know more about the user.
func fnUserData(login string) *UserData {
	// Ignore any error to retrieve the user data: we don't have entries for
	// most of our users, and only store information for those who have
	// particular status (employees and/or maintainers).
	us := &userStore{}
	if ud, err := us.Get(login); err == nil {
		return ud
	}
	return &UserData{Login: login}
}

// fnUserFunction executes an arbitrary binary, passing arbitrary parameters as
// command line arguments.
func fnUserFunction(fullConfig *config.SerializedConfig, binary string) func(...string) (interface{}, error) {
	return func(params ...string) (interface{}, error) {
		cmd := exec.Command(binary, params...)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "ELASTICSEARCH="+fullConfig.ElasticSearch)

		b, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		var i interface{}
		if err := json.Unmarshal(b, &i); err != nil {
			return nil, err
		}
		return i, nil
	}
}
