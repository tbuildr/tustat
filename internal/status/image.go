package status

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// Change is one package change between two deployments.
type Change struct {
	Kind Kind
	Name string
	From string // empty unless the change is an upgrade/downgrade
	To   string
}

// Deployment describes one rpm-ostree deployment.
type Deployment struct {
	Version  string
	Built    time.Time // zero when rpm-ostree reported no timestamp
	Ref      string
	Digest   string
	Checksum string
}

// Image is the booted deployment plus what has been applied or staged.
type Image struct {
	Err           string
	Booted        Deployment
	Applied       *time.Time
	Prev          *Deployment
	Changes       []Change
	DiffErr       string
	Staged        *Deployment
	StagedAt      *time.Time
	StagedChanges []Change
	Check         *string
}

type deployment struct {
	Booted    *bool                      `json:"booted"`
	Staged    *bool                      `json:"staged"`
	Version   *string                    `json:"version"`
	Timestamp *float64                   `json:"timestamp"`
	Ref       *string                    `json:"container-image-reference"`
	Origin    *string                    `json:"origin"`
	Digest    *string                    `json:"container-image-reference-digest"`
	BaseMeta  map[string]json.RawMessage `json:"base-commit-meta"`
	Checksum  *string                    `json:"checksum"`
}

func (d deployment) dep() Deployment {
	var out Deployment
	out.Version = "?"
	if d.Version != nil {
		out.Version = *d.Version
	}
	out.Ref = "?"
	if d.Ref != nil {
		out.Ref = *d.Ref
	} else if d.Origin != nil {
		out.Ref = *d.Origin
	}
	if d.Digest != nil {
		out.Digest = *d.Digest
	}
	if out.Digest == "" && d.BaseMeta != nil {
		if raw, ok := d.BaseMeta["ostree.manifest-digest"]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil {
				out.Digest = s
			}
		}
	}
	if d.Checksum != nil {
		out.Checksum = *d.Checksum
	}
	if d.Timestamp != nil && *d.Timestamp > 0 {
		out.Built = time.Unix(int64(*d.Timestamp), 0)
	}
	return out
}

// GatherImage reads `rpm-ostree status --json`, the journal and optional
// upgrade checks.
func GatherImage(o Options) Image {
	o = o.withDefaults()
	c := &collector{opts: o}
	var r Image

	if !c.have("rpm-ostree") {
		r.Err = "rpm-ostree not found"
		return r
	}
	stdout, _, ok := c.run("rpm-ostree", "status", "--json")
	if !ok {
		r.Err = "rpm-ostree status failed"
		return r
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &root); err != nil {
		r.Err = "couldn't parse rpm-ostree status JSON"
		return r
	}
	raw, present := root["deployments"]
	if !present {
		r.Err = "no deployments in rpm-ostree status"
		return r
	}
	var deps []deployment
	if err := json.Unmarshal(raw, &deps); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		r.Err = "unexpected rpm-ostree status JSON"
		return r
	}

	var booted *Deployment
	for _, d := range deps {
		switch {
		case d.Booted != nil && *d.Booted:
			dep := d.dep()
			booted = &dep
		case d.Staged != nil && *d.Staged:
			dep := d.dep()
			r.Staged = &dep
		case booted != nil && r.Prev == nil:
			dep := d.dep()
			r.Prev = &dep
		}
	}
	if booted == nil {
		r.Err = "no booted deployment found"
		return r
	}
	r.Booted = *booted

	// Applied = staged deployment finalized at shutdown (rpm-ostree and
	// bootc alike).
	r.Applied = c.journalLast("ostree-finalize-staged.service", "Bootloader updated")

	if r.Prev != nil {
		stdout, stderr, ok := c.run("rpm-ostree", "db", "diff", r.Prev.Checksum, r.Booted.Checksum)
		r.Changes = ParseDiff(stdout)
		if !ok {
			if stderr != "" {
				r.DiffErr = stderr
			} else {
				r.DiffErr = "rpm-ostree db diff failed"
			}
		}
	}
	if r.Staged != nil {
		r.StagedAt = c.journalLast("rpm-ostreed.service", `Txn \S+ on \S+ successful`)
		stdout, _, _ := c.run("rpm-ostree", "db", "diff", r.Booted.Checksum, r.Staged.Checksum)
		r.StagedChanges = ParseDiff(stdout)
	}
	if o.Check {
		stdout, stderr, _ := c.run("rpm-ostree", "upgrade", "--check")
		out := stdout
		if out == "" {
			out = stderr
		}
		s := trimOutput(out)
		r.Check = &s
	}
	return r
}

// ParseDiff parses `rpm-ostree db diff` block output.
func ParseDiff(text string) []Change {
	var list []Change
	var kind Kind
	haveKind := false
	for _, raw := range strings.Split(text, "\n") {
		if raw == "" {
			continue
		}
		if raw[0] != ' ' && raw[0] != '\t' {
			switch {
			case strings.HasPrefix(raw, "Upgraded"):
				kind, haveKind = KindUp, true
			case strings.HasPrefix(raw, "Downgraded"):
				kind, haveKind = KindDown, true
			case strings.HasPrefix(raw, "Added"):
				kind, haveKind = KindAdd, true
			case strings.HasPrefix(raw, "Removed"):
				kind, haveKind = KindRemove, true
			default:
				haveKind = false
			}
			continue
		}
		if !haveKind {
			continue
		}
		l := strings.Trim(raw, " \t")
		if arrow := strings.Index(l, " -> "); arrow >= 0 {
			left, to := l[:arrow], l[arrow+4:]
			if sp := strings.LastIndexByte(left, ' '); sp >= 0 {
				list = append(list, Change{Kind: kind, Name: left[:sp], From: left[sp+1:], To: to})
				continue
			}
			// Some rpm-ostree versions print name-version -> version.
			name, from := SplitNEVRA(left)
			list = append(list, Change{Kind: kind, Name: name, From: from, To: stripArch(to)})
			continue
		}
		name, to := SplitNEVRA(l)
		list = append(list, Change{Kind: kind, Name: name, To: to})
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Kind != list[j].Kind {
			return list[i].Kind < list[j].Kind
		}
		return list[i].Name < list[j].Name
	})
	return list
}

var archs = [...]string{"x86_64", "noarch", "i686", "aarch64"}

// stripArch removes a trailing binary architecture from a package
// version-release string.
func stripArch(s string) string {
	if d := strings.LastIndexByte(s, '.'); d >= 0 {
		for _, a := range archs {
			if s[d+1:] == a {
				return s[:d]
			}
		}
	}
	return s
}

// SplitNEVRA splits a name-version-release.arch string into name and
// version-release.
func SplitNEVRA(s string) (name, version string) {
	v := s
	if d := strings.LastIndexByte(v, '.'); d >= 0 {
		for _, a := range archs {
			if v[d+1:] == a {
				v = v[:d]
				break
			}
		}
	}
	r := strings.LastIndexByte(v, '-')
	if r < 0 {
		return s, ""
	}
	n := strings.LastIndexByte(v[:r], '-')
	if n < 0 {
		return v[:r], v[r+1:]
	}
	return v[:n], v[n+1:]
}

// ShortDigest abbreviates a container digest for display.
func ShortDigest(d string) string {
	if d == "" {
		return ""
	}
	colon := strings.IndexByte(d, ':')
	if colon < 0 {
		if len(d) <= 19 {
			return d
		}
		return d[:18] + "…"
	}
	end := colon + 13
	if end > len(d) {
		end = len(d)
	}
	return d[:end]
}
