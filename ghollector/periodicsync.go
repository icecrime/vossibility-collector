package main

import (
	"fmt"
	"time"
)

const (
	SyncHourly = "hourly"
	SyncDaily  = "daily"
	SyncWeekly = "weekly"
)

var nextTickTime = map[string]func(time.Time) time.Duration{
	SyncHourly: nextHourlyTick,
	SyncDaily:  nextDailyTick,
	SyncWeekly: nextWeeklyTick,
}

type PeriodicSync string

func NewPeriodicSync(v string) (PeriodicSync, error) {
	p := PeriodicSync(v)
	if !p.IsValid() {
		return "", fmt.Errorf("invalid value %q for sync periodicity", v)
	}
	return p, nil
}

func (p PeriodicSync) IsValid() bool {
	_, ok := nextTickTime[string(p)]
	return ok
}

// Next calculates the time until the next full synchronization.  We could
// argue that this would be better achived with cron: I'd like the app to know
// about that frequency, it makes the whole event persistence easier.
//
// If the process is stopped, we will miss some ticks, and we won't try to
// catch up: I have no idea if this is going to be a problem later down to the
// road when querying.
func (p PeriodicSync) Next() time.Duration {
	now := time.Now()
	if f, ok := nextTickTime[string(p)]; ok {
		return f(now)
	}
	return 0
}

func nextHourlyTick(ref time.Time) time.Duration {
	return time.Duration(60-ref.Minute()-1)*time.Minute + time.Duration(60-ref.Second())*time.Second
}

func nextDailyTick(ref time.Time) time.Duration {
	return time.Duration(24-ref.Hour()-1)*time.Hour + nextHourlyTick(ref)
}

func nextWeeklyTick(ref time.Time) time.Duration {
	return time.Duration(7-int(ref.Weekday())-1)*24*time.Hour + nextDailyTick(ref)
}
