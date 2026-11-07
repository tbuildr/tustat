package status

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tustat/internal/timefmt"
)

// AppImage is one AppImage file on disk.
type AppImage struct {
	Path    string
	File    string
	Name    string
	Version string
	MTime   time.Time
	Size    int64
}

// AppImageEvent is one line of the AppImage change log.
type AppImageEvent struct {
	Stamp  string
	Kind   string // added, updated or removed
	File   string
	Detail string
}

// AppImages is the scan result plus the persisted change history.
type AppImages struct {
	Items     []AppImage
	Dirs      []string
	FirstRun  bool
	LogPath   string
	LogEvents []AppImageEvent
	Check     *string
}

// GatherAppImages scans the AppImage directories and diffs them against
// the snapshot kept in the state directory.
func GatherAppImages(o Options) AppImages {
	o = o.withDefaults()
	c := &collector{opts: o}
	var r AppImages

	dirsEnv := o.Getenv("TUSTAT_APPIMAGE_DIRS")
	if dirsEnv == "" {
		dirsEnv = fmt.Sprintf("%s/AppImages:%s/Applications:%s/.local/bin", o.Home, o.Home, o.Home)
	}
	for _, d := range strings.Split(dirsEnv, ":") {
		if d != "" {
			r.Dirs = append(r.Dirs, d)
		}
	}

	var items []AppImage
	for _, d := range r.Dirs {
		scanAppImages(&items, d, 1)
	}

	// Desktop entries give the human name and version (Gear Lever writes
	// X-AppImage-Version).
	desktops := readDesktopEntries(filepath.Join(o.Home, ".local", "share", "applications"))
	for i := range items {
		items[i].Name = ""
		items[i].Version = "-"
		for _, content := range desktops {
			if !strings.Contains(content, items[i].Path) {
				continue
			}
			items[i].Name = lineVal(content, "Name=")
			if v := lineVal(content, "X-AppImage-Version="); v != "" {
				items[i].Version = v
			}
			break
		}
		if items[i].Name == "" {
			items[i].Name = items[i].File
		}
	}

	stateDir := filepath.Join(o.Home, ".local", "state", "tustat")
	if x := o.Getenv("XDG_STATE_HOME"); x != "" {
		stateDir = filepath.Join(x, "tustat")
	}
	_ = os.MkdirAll(stateDir, 0o755)
	snapPath := filepath.Join(stateDir, "appimages.tsv")
	r.LogPath = filepath.Join(stateDir, "appimage-history.log")

	type oldEntry struct {
		mtime, size, version string
		seen                 bool
	}
	old := make(map[string]*oldEntry)
	if data, err := os.ReadFile(snapPath); err != nil {
		r.FirstRun = true
	} else {
		for _, line := range strings.Split(string(data), "\n") {
			f := strings.Split(line, "\t")
			if len(f) == 0 || f[0] == "" {
				continue
			}
			e := &oldEntry{version: "-"}
			if len(f) > 1 {
				e.mtime = f[1]
			}
			if len(f) > 2 {
				e.size = f[2]
			}
			if len(f) > 3 {
				e.version = f[3]
			}
			old[f[0]] = e
		}
	}

	var snapNew, logAdd strings.Builder
	for _, a := range items {
		fmt.Fprintf(&snapNew, "%s\t%d\t%d\t%s\n", a.Path, a.MTime.Unix(), a.Size, a.Version)
		if r.FirstRun {
			continue
		}
		if prev, ok := old[a.Path]; ok {
			prev.seen = true
			same := prev.mtime == fmt.Sprint(a.MTime.Unix()) && prev.size == fmt.Sprint(a.Size)
			if !same {
				fmt.Fprintf(&logAdd, "%s\tupdated\t%s\t%s -> %s\n", timefmt.Stamp(a.MTime), a.File, prev.version, a.Version)
			}
		} else {
			fmt.Fprintf(&logAdd, "%s\tadded\t%s\t%s\n", timefmt.Stamp(a.MTime), a.File, a.Version)
		}
	}
	if !r.FirstRun {
		gone := make([]string, 0, len(old))
		for path, prev := range old {
			if !prev.seen {
				gone = append(gone, path)
			}
		}
		sort.Strings(gone)
		for _, path := range gone {
			fmt.Fprintf(&logAdd, "%s\tremoved\t%s\t%s\n", timefmt.Stamp(o.Now), filepath.Base(path), old[path].version)
		}
	}
	_ = os.WriteFile(snapPath, []byte(snapNew.String()), 0o644)
	if logAdd.Len() > 0 {
		appendFile(r.LogPath, logAdd.String())
	}

	if data, err := os.ReadFile(r.LogPath); err == nil {
		var lines []string
		for _, line := range strings.Split(string(data), "\n") {
			if line != "" {
				lines = append(lines, line)
			}
		}
		keep := 8
		if o.All {
			keep = len(lines)
		}
		if start := len(lines) - keep; start > 0 {
			lines = lines[start:]
		}
		for _, line := range lines {
			f := strings.Split(line, "\t")
			ev := AppImageEvent{Stamp: f[0]}
			if len(f) > 1 {
				ev.Kind = f[1]
			}
			if len(f) > 2 {
				ev.File = f[2]
			}
			if len(f) > 3 {
				ev.Detail = f[3]
			}
			r.LogEvents = append(r.LogEvents, ev)
		}
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].MTime.After(items[j].MTime) })
	r.Items = items

	if o.Check {
		stdout, _, ok := c.run("flatpak", "run", "it.mijorus.gearlever", "--list-updates")
		s := "Gear Lever not available — can't check"
		if ok {
			s = trimOutput(stdout)
		}
		r.Check = &s
	}
	return r
}

func scanAppImages(list *[]AppImage, path string, depth int) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, e := range entries {
		full := filepath.Join(path, e.Name())
		switch {
		case e.IsDir():
			if depth > 0 {
				scanAppImages(list, full, depth-1)
			}
		case e.Type().IsRegular() || e.Type()&os.ModeSymlink != 0:
			if !strings.HasSuffix(strings.ToLower(e.Name()), ".appimage") {
				continue
			}
			info, err := os.Stat(full)
			if err != nil {
				continue
			}
			*list = append(*list, AppImage{
				Path:    full,
				File:    e.Name(),
				Version: "-",
				MTime:   info.ModTime(),
				Size:    info.Size(),
			})
		}
	}
}

const desktopLimit = 256 << 10

func readDesktopEntries(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil || len(data) > desktopLimit {
			continue
		}
		out = append(out, string(data))
	}
	return out
}

// lineVal returns the value of the first line starting with key.
func lineVal(content, key string) string {
	for _, l := range strings.Split(content, "\n") {
		if strings.HasPrefix(l, key) {
			return strings.Trim(l[len(key):], " \r")
		}
	}
	return ""
}

func appendFile(path, data string) {
	old, _ := os.ReadFile(path)
	_ = os.WriteFile(path, append(old, data...), 0o644)
}
