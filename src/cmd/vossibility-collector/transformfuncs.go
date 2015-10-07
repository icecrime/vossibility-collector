package main

import "time"

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

func fnIdentity(v interface{}) func() interface{} {
	return func() interface{} {
		return v
	}
}

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
