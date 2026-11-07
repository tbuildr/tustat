package status

import "time"

// Timer is one automatic-update systemd timer.
type Timer struct {
	Name    string
	User    bool
	Enabled string
	Last    *time.Time
	Next    *time.Time
	Result  string
}

var timerSpecs = [...]struct {
	name string
	user bool
}{
	{"uupd.timer", false},
	{"ublue-update.timer", false},
	{"rpm-ostreed-automatic.timer", false},
	{"flatpak-system-update.timer", false},
	{"flatpak-user-update.timer", true},
}

// GatherTimers inspects the update timers with systemctl.
func GatherTimers(o Options) []Timer {
	c := &collector{opts: o}
	var list []Timer
	for _, spec := range timerSpecs {
		base := []string{"systemctl"}
		if spec.user {
			base = []string{"systemctl", "--user"}
		}
		args := append(append([]string{}, base[1:]...),
			"show", "--timestamp=unix", "-p", "LoadState", "-p", "UnitFileState",
			"-p", "LastTriggerUSec", "-p", "NextElapseUSecRealtime", "-p", "Unit", spec.name)
		stdout, _, _ := c.run(base[0], args...)
		if prop(stdout, "LoadState") != "loaded" {
			continue
		}
		svc := prop(stdout, "Unit")
		result := "?"
		if svc != "" {
			resultArgs := append(append([]string{}, base[1:]...), "show", "-P", "Result", svc)
			out, _, _ := c.run(base[0], resultArgs...)
			result = trimOutput(out)
		}
		list = append(list, Timer{
			Name:    spec.name,
			User:    spec.user,
			Enabled: prop(stdout, "UnitFileState"),
			Last:    unixProp(prop(stdout, "LastTriggerUSec")),
			Next:    unixProp(prop(stdout, "NextElapseUSecRealtime")),
			Result:  result,
		})
	}
	return list
}

// TimersFailing counts timers whose last run did not succeed.
func TimersFailing(timers []Timer) int {
	n := 0
	for _, t := range timers {
		if t.Last != nil && t.Result != "success" {
			n++
		}
	}
	return n
}

// LatestRun returns the most recent Last across timers.
func LatestRun(timers []Timer) *time.Time {
	var latest *time.Time
	for _, t := range timers {
		if t.Last == nil {
			continue
		}
		if latest == nil || t.Last.After(*latest) {
			latest = t.Last
		}
	}
	return latest
}
