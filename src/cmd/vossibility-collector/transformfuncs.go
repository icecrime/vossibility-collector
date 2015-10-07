package main

import "time"

// fnContext is a constant function passed down to templates as a mean to
// access contextual information on the data being transformed (e.g.,
// repository information).
func fnContext(context Context) func() interface{} {
	return func() interface{} {
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
func fnUserData(login string) interface{} {
	// Ignore any error to retrieve the user data: we don't have entries for
	// most of our users, and only store information for those who have
	// particular status (employees and/or maintainers).
	us := &userStore{}
	if ud, err := us.Get(login); err == nil {
		return ud
	}
	return &UserData{Login: login}
}
