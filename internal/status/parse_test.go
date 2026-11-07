package status

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSplitNEVRA(t *testing.T) {
	cases := []struct{ in, name, version string }{
		{"bash-5.2.37-1.fc41.x86_64", "bash", "5.2.37-1.fc41"},
		{"bash-5.2.37-1.fc41", "bash", "5.2.37-1.fc41"},
		{"nosep", "nosep", ""},
		{"aarch64-thing-1.0-2.aarch64", "aarch64-thing", "1.0-2"},
	}
	for _, c := range cases {
		name, version := SplitNEVRA(c.in)
		if name != c.name || version != c.version {
			t.Errorf("SplitNEVRA(%q) = %q, %q; want %q, %q", c.in, name, version, c.name, c.version)
		}
	}
}

func TestParseDiff(t *testing.T) {
	text := `Upgraded:
  bash 5.2.37-1.fc41 -> 5.2.38-1.fc41
  kernel-6.11.5-300.fc41.x86_64 -> 6.11.6-300.fc41.x86_64
Added:
  newpkg-1.0-1.fc41.x86_64
Removed:
  oldpkg-2.0-3.fc41.x86_64
Downgraded:
  foo 2.0-1 -> 1.9-1
`
	got := ParseDiff(text)
	want := []Change{
		{KindUp, "bash", "5.2.37-1.fc41", "5.2.38-1.fc41"},
		{KindUp, "kernel", "6.11.5-300.fc41", "6.11.6-300.fc41"},
		{KindDown, "foo", "2.0-1", "1.9-1"},
		{KindAdd, "newpkg", "", "1.0-1.fc41"},
		{KindRemove, "oldpkg", "", "2.0-3.fc41"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d changes, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("change %d = %#v; want %#v", i, got[i], want[i])
		}
	}
}

func TestParseDiffIgnoresPreambleLines(t *testing.T) {
	got := ParseDiff("No differences found\n")
	if len(got) != 0 {
		t.Fatalf("got %#v; want no changes", got)
	}
}

func TestParseFlatTime(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.Local)

	got := parseFlatTime("2026-10-14 10:23:01", now)
	want := time.Date(2026, 10, 14, 10, 23, 1, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("numeric time = %v; want %v", got, want)
	}

	got = parseFlatTime("Nov 6 09:15:02", now)
	want = time.Date(2025, 11, 6, 9, 15, 2, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("month-name time in the future = %v; want previous year %v", got, want)
	}

	now = time.Date(2026, 11, 7, 12, 0, 0, 0, time.Local)
	got = parseFlatTime("Oct 14 10:23:01", now)
	want = time.Date(2026, 10, 14, 10, 23, 1, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("month-name time = %v; want %v", got, want)
	}

	if got := parseFlatTime("garbage", now); !got.IsZero() {
		t.Errorf("unparsable time = %v; want zero", got)
	}
}

func TestSortEntries(t *testing.T) {
	entries := []FlatpakEntry{
		{Day: "2026-10-20", T: time.Date(2026, 10, 20, 11, 0, 0, 0, time.Local)},
		{Day: "2026-10-12", T: time.Date(2026, 10, 12, 9, 0, 0, 0, time.Local)},
		{Day: "unknown"},
		{Day: "2026-10-14", T: time.Date(2026, 10, 14, 10, 0, 0, 0, time.Local)},
	}
	sortEntries(entries)
	got := []string{entries[0].Day, entries[1].Day, entries[2].Day, entries[3].Day}
	want := []string{"unknown", "2026-10-12", "2026-10-14", "2026-10-20"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v; want %v", got, want)
		}
	}
}

func TestShortDigest(t *testing.T) {
	if got := ShortDigest("sha256:abcdef0123456789"); got != "sha256:abcdef012345" {
		t.Errorf("ShortDigest = %q", got)
	}
	if got := ShortDigest(""); got != "" {
		t.Errorf("ShortDigest(empty) = %q", got)
	}
}

// fakeEnv returns an Options whose PATH and environment point at a test
// fixture directory.
func fakeEnv(t *testing.T, binDir, home string) Options {
	t.Helper()
	return Options{
		Days: 30,
		Now:  time.Date(2026, 11, 7, 12, 0, 0, 0, time.Local),
		Home: home,
		Getenv: func(k string) string {
			switch k {
			case "PATH":
				return binDir + ":" + os.Getenv("PATH")
			case "HOME":
				return home
			case "XDG_STATE_HOME":
				return filepath.Join(home, "state")
			}
			return ""
		},
	}
}

func writeScript(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}
