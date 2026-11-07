package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"tustat/internal/status"
)

func testView() *View {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.Local)
	return NewView(NewTheme(lipgloss.NewRenderer(&bytes.Buffer{})), now, "/home/u", false)
}

func testData() status.Data {
	applied := time.Date(2026, 11, 6, 22, 0, 0, 0, time.Local)
	stagedAt := time.Date(2026, 11, 6, 10, 0, 0, 0, time.Local)
	staged := status.Deployment{Version: "41.20251105.0"}
	return status.Data{
		Image: status.Image{
			Booted: status.Deployment{
				Version: "41.20251101.0",
				Built:   time.Date(2026, 11, 1, 9, 0, 0, 0, time.Local),
				Ref:     "ostree-image-signed:docker://ghcr.io/ublue-os/bazzite-gnome:stable",
				Digest:  "sha256:abcdef0123456789",
			},
			Prev:    &status.Deployment{Version: "41.20251020.0"},
			Applied: &applied,
			Changes: []status.Change{
				{Kind: status.KindUp, Name: "bash", From: "5.2.37-1.fc41", To: "5.2.38-1.fc41"},
				{Kind: status.KindAdd, Name: "newpkg", To: "1.0-1.fc41"},
			},
			Staged:   &staged,
			StagedAt: &stagedAt,
		},
		Flatpak: status.Flatpak{
			Days: 30,
			Entries: []status.FlatpakEntry{
				{T: time.Date(2026, 10, 20, 11, 5, 0, 0, time.Local), Day: "2026-10-20", Kind: status.KindUp,
					App: "org.example.App", Branch: "stable", Inst: "system", Clock: "11:05"},
			},
			Versions: map[string]string{"org.example.App": "1.2.3"},
		},
		AppImages: status.AppImages{
			Items: []status.AppImage{
				{Path: "/home/u/AppImages/alpha.AppImage", File: "alpha.AppImage", Name: "Alpha",
					Version: "1.0", MTime: time.Date(2026, 11, 1, 10, 0, 0, 0, time.Local)},
			},
			Dirs:    []string{"/home/u/AppImages"},
			LogPath: "/home/u/.local/state/tustat/appimage-history.log",
		},
		Timers: []status.Timer{
			{Name: "uupd.timer", Enabled: "enabled", Result: "success"},
		},
	}
}

func TestCardsAreExactlyTheRequestedWidth(t *testing.T) {
	v := testView()
	data := testData()
	for _, width := range []int{40, 80, 100} {
		for _, s := range status.Sections() {
			for _, line := range v.Section(s, data, width) {
				if got := ansi.StringWidth(line); got != width {
					t.Errorf("%s width %d: line %q has width %d", s.Name(), width, line, got)
				}
			}
		}
	}
}

func TestErrorCards(t *testing.T) {
	v := testView()
	data := status.Data{
		Image:     status.Image{Err: "rpm-ostree not found"},
		Flatpak:   status.Flatpak{Err: "flatpak not installed"},
		AppImages: status.AppImages{},
	}
	var all strings.Builder
	for _, s := range status.Sections() {
		all.WriteString(strings.Join(v.Section(s, data, 80), "\n") + "\n")
	}
	out := all.String()
	for _, want := range []string{"rpm-ostree not found", "flatpak not installed", "no AppImages in", "no update timers installed"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestSummaryRowFitsWidth(t *testing.T) {
	v := testView()
	data := testData()
	for _, width := range []int{40, 80} {
		for _, selected := range []bool{false, true} {
			for _, s := range status.Sections() {
				row := v.SummaryRow(s, status.Summarize(s, data), 1, selected, width)
				if got := ansi.StringWidth(row); got > width {
					t.Errorf("summary row width %d > %d: %q", got, width, row)
				}
				if !strings.Contains(row, s.Name()) {
					t.Errorf("summary row missing %q: %q", s.Name(), row)
				}
			}
		}
	}
}

func TestSummaryUsesRecentState(t *testing.T) {
	data := testData()
	im := status.Summarize(status.SectionImage, data)
	if im.T == nil || !strings.Contains(im.Detail, "staged") {
		t.Errorf("image summary = %+v", im)
	}
	fp := status.Summarize(status.SectionFlatpak, data)
	if fp.T == nil || fp.Detail != "1 change in last run" {
		t.Errorf("flatpak summary = %+v", fp)
	}
	ai := status.Summarize(status.SectionAppImage, data)
	if ai.Detail != "1 app" {
		t.Errorf("appimage summary = %+v", ai)
	}
	tm := status.Summarize(status.SectionTimers, data)
	if tm.Tone != status.ToneOK || tm.Detail != "✓ all ok" {
		t.Errorf("timers summary = %+v", tm)
	}
}

func TestSplitNEVRADisplay(t *testing.T) {
	// Guard the shape used by the change list: name, then version columns.
	v := testView()
	lines := v.changes([]status.Change{{Kind: status.KindUp, Name: "bash", From: "1", To: "2"}}, "", 80)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "bash") || !strings.Contains(joined, "→") {
		t.Errorf("change lines = %q", joined)
	}
}
