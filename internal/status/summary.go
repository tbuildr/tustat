package status

import (
	"fmt"
	"time"
)

// Summary is the one-line state of a section, shown in menus and print
// headers.
type Summary struct {
	T      *time.Time
	Detail string
	Tone   Tone
}

// Summarize condenses a gathered snapshot into a menu line.
func Summarize(s Section, d Data) Summary {
	switch s {
	case SectionImage:
		x := d.Image
		if x.Err != "" {
			return Summary{Detail: x.Err, Tone: ToneWarn}
		}
		if x.Staged != nil {
			return Summary{
				T:      x.Applied,
				Detail: fmt.Sprintf("%s  ⟳ %s staged — reboot", x.Booted.Version, x.Staged.Version),
				Tone:   ToneWarn,
			}
		}
		return Summary{T: x.Applied, Detail: x.Booted.Version + "  ✓ nothing pending"}

	case SectionFlatpak:
		x := d.Flatpak
		if x.Err != "" {
			return Summary{Detail: x.Err, Tone: ToneWarn}
		}
		if len(x.Entries) == 0 {
			return Summary{Detail: fmt.Sprintf("no changes in %d days", x.Days), Tone: ToneDim}
		}
		last := x.Entries[len(x.Entries)-1]
		n := 0
		for _, e := range x.Entries {
			if e.Day == last.Day {
				n++
			}
		}
		s := Summary{Detail: fmt.Sprintf("%d change%s in last run", n, plural(n)), Tone: TonePlain}
		if !last.T.IsZero() {
			t := last.T
			s.T = &t
		}
		return s

	case SectionAppImage:
		x := d.AppImages
		if len(x.Items) == 0 {
			return Summary{Detail: "none found", Tone: ToneDim}
		}
		t := x.Items[0].MTime
		n := len(x.Items)
		return Summary{T: &t, Detail: fmt.Sprintf("%d app%s", n, plural(n))}

	default:
		x := d.Timers
		if len(x) == 0 {
			return Summary{Detail: "none installed", Tone: ToneDim}
		}
		latest := LatestRun(x)
		if n := TimersFailing(x); n > 0 {
			return Summary{T: latest, Detail: fmt.Sprintf("✗ %d failing", n), Tone: ToneDanger}
		}
		return Summary{T: latest, Detail: "✓ all ok", Tone: ToneOK}
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
