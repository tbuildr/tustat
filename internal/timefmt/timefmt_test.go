package timefmt

import (
	"testing"
	"time"
)

func date(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.Local)
}

func TestWhen(t *testing.T) {
	now := date(2026, time.November, 7, 12, 0)
	same := date(2026, time.November, 6, 22, 5)
	if got, want := When(now, same), "Fri 6 Nov 22:05"; got != want {
		t.Errorf("When(same year) = %q; want %q", got, want)
	}
	older := date(2025, time.November, 1, 9, 0)
	if got, want := When(now, older), "1 Nov 2025 09:00"; got != want {
		t.Errorf("When(other year) = %q; want %q", got, want)
	}
	if got, want := DayLabel(date(2026, time.October, 20, 11, 5)), "Tue 20 Oct"; got != want {
		t.Errorf("DayLabel = %q; want %q", got, want)
	}
}

func TestSpan(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Minute, "30 min"},
		{3 * time.Hour, "3h"},
		{24 * time.Hour, "1 day"},
		{5 * 24 * time.Hour, "5 days"},
		{20 * 24 * time.Hour, "2 weeks"},
		{90 * 24 * time.Hour, "3 months"},
	}
	for _, c := range cases {
		if got := Span(c.d); got != c.want {
			t.Errorf("Span(%v) = %q; want %q", c.d, got, c.want)
		}
	}
}

func TestAgo(t *testing.T) {
	now := date(2026, time.November, 7, 12, 0)
	if got := Ago(now, now.Add(-30*time.Second)); got != "just now" {
		t.Errorf("Ago(recent) = %q", got)
	}
	if got := Ago(now, now.Add(-3*24*time.Hour)); got != "3 days ago" {
		t.Errorf("Ago(past) = %q", got)
	}
	if got := Ago(now, now.Add(2*time.Hour)); got != "in 2h" {
		t.Errorf("Ago(future) = %q", got)
	}
}

func TestAgeTier(t *testing.T) {
	now := date(2026, time.November, 7, 12, 0)
	cases := []struct {
		t    time.Time
		want Tier
	}{
		{now.Add(-6 * 24 * time.Hour), TierFresh},
		{now.Add(-10 * 24 * time.Hour), TierAgeing},
		{now.Add(-40 * 24 * time.Hour), TierStale},
	}
	for _, c := range cases {
		if got := AgeTier(now, c.t); got != c.want {
			t.Errorf("AgeTier(%v) = %v; want %v", c.t, got, c.want)
		}
	}
}

func TestStamp(t *testing.T) {
	if got, want := Stamp(date(2026, time.November, 6, 22, 5)), "2026-11-06 22:05"; got != want {
		t.Errorf("Stamp = %q; want %q", got, want)
	}
}
