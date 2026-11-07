package status

import (
	"sort"
	"strings"
	"time"
)

// FlatpakEntry is one line of `flatpak history`.
type FlatpakEntry struct {
	T      time.Time
	Day    string
	Kind   Kind
	App    string
	Branch string
	Inst   string
	Clock  string
}

// Flatpak is the Flatpak history window plus installed versions.
type Flatpak struct {
	Err      string
	Days     int
	Entries  []FlatpakEntry
	Versions map[string]string
	Check    *string
}

// GatherFlatpak reads `flatpak history` and `flatpak list`.
func GatherFlatpak(o Options) Flatpak {
	o = o.withDefaults()
	c := &collector{opts: o}
	r := Flatpak{Days: o.Days}
	if !c.have("flatpak") {
		r.Err = "flatpak not installed"
		return r
	}

	since := o.Now.Add(-time.Duration(o.Days) * 24 * time.Hour)
	sinceArg := "--since=" + since.Format("2006-01-02 15:04:05")
	stdout, _, _ := c.run("flatpak", "history", sinceArg, "--columns=time,change,application,branch,installation")

	for _, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 2 {
			continue
		}
		timeField, change := f[0], f[1]
		var kind Kind
		switch {
		case strings.Contains(change, "uninstall"):
			kind = KindRemove
		case strings.Contains(change, "update"):
			kind = KindUp
		case strings.Contains(change, "install"):
			kind = KindAdd
		default:
			continue
		}
		field := func(i int) string {
			if i < len(f) {
				return strings.Trim(f[i], " ")
			}
			return ""
		}
		t := parseFlatTime(timeField, o.Now)
		clock := timeField
		if i := strings.LastIndexByte(timeField, ' '); i >= 0 {
			clock = timeField[i+1:]
		}
		if len(clock) > 5 {
			clock = clock[:5]
		}
		r.Entries = append(r.Entries, FlatpakEntry{
			T:      t,
			Day:    dayKey(t, timeField),
			Kind:   kind,
			App:    field(2),
			Branch: field(3),
			Inst:   field(4),
			Clock:  clock,
		})
	}
	// flatpak history is not guaranteed to be ordered; the newest entry
	// drives "last run", so sort chronologically while keeping entries
	// with unparsable timestamps in their original order.
	sortEntries(r.Entries)

	stdout, _, _ = c.run("flatpak", "list", "--columns=application,version")
	r.Versions = make(map[string]string)
	for _, line := range strings.Split(stdout, "\n") {
		f := strings.Split(line, "\t")
		app := f[0]
		ver := ""
		if len(f) > 1 {
			ver = f[1]
		}
		if app != "" && ver != "" {
			r.Versions[app] = ver
		}
	}

	if o.Check {
		stdout, _, _ := c.run("flatpak", "remote-ls", "--updates", "--columns=application,version,branch")
		s := trimOutput(stdout)
		r.Check = &s
	}
	return r
}

func sortEntries(entries []FlatpakEntry) {
	// Undated entries sort first so the newest dated entry stays last; the
	// summary treats the last entry as the most recent run.
	undated := make([]FlatpakEntry, 0, len(entries))
	dated := make([]FlatpakEntry, 0, len(entries))
	for _, e := range entries {
		if e.T.IsZero() {
			undated = append(undated, e)
		} else {
			dated = append(dated, e)
		}
	}
	sort.SliceStable(dated, func(i, j int) bool { return dated[i].T.Before(dated[j].T) })
	copy(entries, append(undated, dated...))
}

var monthNames = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// parseFlatTime accepts "Oct 14 10:23:01" (no year; assumes the most
// recent past year) or "2026-10-14 10:23:01".
func parseFlatTime(s string, now time.Time) time.Time {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '-' || r == ':' || r == 'T'
	})
	atoi := func(i int) int {
		if i >= len(fields) {
			return 0
		}
		n := 0
		for _, ch := range fields[i] {
			if ch < '0' || ch > '9' {
				return 0
			}
			n = n*10 + int(ch-'0')
		}
		return n
	}
	if len(s) >= 10 && s[0] >= '0' && s[0] <= '9' {
		return time.Date(atoi(0), time.Month(atoi(1)), atoi(2), atoi(3), atoi(4), atoi(5), 0, time.Local)
	}
	if len(fields) == 0 {
		return time.Time{}
	}
	mon := -1
	for i, m := range monthNames {
		if strings.EqualFold(fields[0], m) {
			mon = i
			break
		}
	}
	if mon < 0 {
		return time.Time{}
	}
	year := now.In(time.Local).Year()
	t := time.Date(year, time.Month(mon+1), atoi(1), atoi(2), atoi(3), atoi(4), 0, time.Local)
	if t.After(now.Add(24 * time.Hour)) {
		t = time.Date(year-1, time.Month(mon+1), atoi(1), atoi(2), atoi(3), atoi(4), 0, time.Local)
	}
	return t
}

// dayKey groups entries by local calendar day, falling back to the raw
// date field when the timestamp could not be parsed.
func dayKey(t time.Time, raw string) string {
	if !t.IsZero() {
		return t.Local().Format("2006-01-02")
	}
	if i := strings.LastIndexByte(raw, ' '); i >= 0 {
		return raw[:i]
	}
	return raw
}
