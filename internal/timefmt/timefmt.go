// Package timefmt renders local wall-clock times and human ages the way
// tustat has always printed them.
package timefmt

import (
	"fmt"
	"time"
)

var (
	weekdays = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	months   = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
)

// When renders a timestamp: "Mon 3 Nov 14:05" inside the current year,
// "3 Nov 2026 14:05" otherwise.
func When(now, t time.Time) string {
	t = t.Local()
	if t.Year() == now.Local().Year() {
		return fmt.Sprintf("%s %d %s %02d:%02d", weekdays[t.Weekday()], t.Day(), months[int(t.Month())-1], t.Hour(), t.Minute())
	}
	return fmt.Sprintf("%d %s %d %02d:%02d", t.Day(), months[int(t.Month())-1], t.Year(), t.Hour(), t.Minute())
}

// DayLabel renders a day without a year: "Mon 3 Nov".
func DayLabel(t time.Time) string {
	t = t.Local()
	return fmt.Sprintf("%s %d %s", weekdays[t.Weekday()], t.Day(), months[int(t.Month())-1])
}

// Span renders a duration compactly: "12 min", "5h", "3 days", "4 weeks",
// "7 months".
func Span(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	days := int(d / (24 * time.Hour))
	if days < 14 {
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	if days < 60 {
		return fmt.Sprintf("%d weeks", days/7)
	}
	return fmt.Sprintf("%d months", days/30)
}

// Ago renders a timestamp relative to now: "just now", "2h ago" or
// "in 3h" for future timestamps.
func Ago(now, t time.Time) string {
	d := now.Sub(t)
	if d < -time.Minute {
		return "in " + Span(-d)
	}
	if d < time.Minute {
		return "just now"
	}
	return Span(d) + " ago"
}

// Tier buckets an age for colouring: fresh, ageing or stale.
type Tier int

const (
	TierFresh  Tier = iota // under a week
	TierAgeing             // under a month
	TierStale              // a month or more
)

// AgeTier classifies how old t is.
func AgeTier(now, t time.Time) Tier {
	d := now.Sub(t)
	switch {
	case d < 7*24*time.Hour:
		return TierFresh
	case d < 30*24*time.Hour:
		return TierAgeing
	}
	return TierStale
}

// Stamp renders the timestamp used in the AppImage history log.
func Stamp(t time.Time) string {
	t = t.Local()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d", t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute())
}
