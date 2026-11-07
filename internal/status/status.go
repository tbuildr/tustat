// Package status gathers update status for vertos: the booted ostree image,
// Flatpak history, AppImages on disk and the automatic-update timers.
//
// It is read-only apart from the AppImage snapshot and change log under
// ${XDG_STATE_HOME:-~/.local/state}/tustat, which keep the same format as
// the original bash and Zig implementations.
package status

import (
	"os"
	"strings"
	"time"
)

// Kind is the direction of a package change.
type Kind int

const (
	KindUp Kind = iota
	KindDown
	KindAdd
	KindRemove
)

// Marker is the single-glyph indicator used in change lists.
func (k Kind) Marker() string {
	switch k {
	case KindUp:
		return "↑"
	case KindDown:
		return "↓"
	case KindAdd:
		return "+"
	default:
		return "−"
	}
}

// Label is the plural noun used in change summaries.
func (k Kind) Label() string {
	switch k {
	case KindUp:
		return "upgraded"
	case KindDown:
		return "downgraded"
	case KindAdd:
		return "added"
	default:
		return "removed"
	}
}

// Section identifies one status panel.
type Section int

const (
	SectionImage Section = iota
	SectionFlatpak
	SectionAppImage
	SectionTimers
)

var allSections = [...]Section{SectionImage, SectionFlatpak, SectionAppImage, SectionTimers}

// Sections returns the sections in display order.
func Sections() []Section { return allSections[:] }

// Name is the display name of a section.
func (s Section) Name() string {
	switch s {
	case SectionImage:
		return "Image"
	case SectionFlatpak:
		return "Flatpak"
	case SectionAppImage:
		return "AppImage"
	default:
		return "Auto-updates"
	}
}

// Icon is the glyph shown next to a section.
func (s Section) Icon() string {
	switch s {
	case SectionImage:
		return "◆"
	case SectionFlatpak:
		return "▣"
	case SectionAppImage:
		return "◇"
	default:
		return "◷"
	}
}

// ParseSection maps a CLI argument to a section.
func ParseSection(s string) (Section, bool) {
	switch strings.ToLower(s) {
	case "image":
		return SectionImage, true
	case "flatpak":
		return SectionFlatpak, true
	case "appimage":
		return SectionAppImage, true
	case "timers":
		return SectionTimers, true
	}
	return 0, false
}

// Tone describes how a value should be coloured by the renderer.
type Tone int

const (
	TonePlain Tone = iota
	ToneOK
	ToneWarn
	ToneDanger
	ToneDim
)

// Options drives a gather. Zero values are filled in by withDefaults.
type Options struct {
	Check  bool // also check what updates are available (network)
	All    bool // show every item, not just the latest
	Days   int  // Flatpak history window
	Now    time.Time
	Home   string
	Getenv func(string) string
}

// DefaultOptions returns the environment-derived defaults.
func DefaultOptions() Options {
	return Options{
		Days:   30,
		Now:    time.Now(),
		Home:   os.Getenv("HOME"),
		Getenv: os.Getenv,
	}
}

func (o Options) withDefaults() Options {
	if o.Days < 0 {
		o.Days = 30
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	if o.Getenv == nil {
		o.Getenv = os.Getenv
	}
	if o.Home == "" {
		o.Home = o.Getenv("HOME")
	}
	return o
}

// Data is one gathered snapshot of every section.
type Data struct {
	Image     Image
	Flatpak   Flatpak
	AppImages AppImages
	Timers    []Timer
}

// GatherAll collects every section.
func GatherAll(o Options) Data {
	o = o.withDefaults()
	return Data{
		Image:     GatherImage(o),
		Flatpak:   GatherFlatpak(o),
		AppImages: GatherAppImages(o),
		Timers:    GatherTimers(o),
	}
}

// GatherSelected collects only the requested sections.
func GatherSelected(o Options, secs []Section) Data {
	o = o.withDefaults()
	var d Data
	for _, s := range secs {
		d = d.Merge(s, gatherOne(o, s))
	}
	return d
}

func gatherOne(o Options, s Section) Data {
	var d Data
	switch s {
	case SectionImage:
		d.Image = GatherImage(o)
	case SectionFlatpak:
		d.Flatpak = GatherFlatpak(o)
	case SectionAppImage:
		d.AppImages = GatherAppImages(o)
	case SectionTimers:
		d.Timers = GatherTimers(o)
	}
	return d
}

// Merge replaces the given section of d with the one from other.
func (d Data) Merge(s Section, other Data) Data {
	switch s {
	case SectionImage:
		d.Image = other.Image
	case SectionFlatpak:
		d.Flatpak = other.Flatpak
	case SectionAppImage:
		d.AppImages = other.AppImages
	case SectionTimers:
		d.Timers = other.Timers
	}
	return d
}
