package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"tustat/internal/status"
)

func testModel(t *testing.T) model {
	t.Helper()
	opts := status.DefaultOptions()
	opts.Now = time.Date(2026, 11, 7, 12, 0, 0, 0, time.Local)
	m := newModel(Config{Options: opts, Version: "test"})
	m.width, m.height = 100, 40
	m.loaded = true
	m.screen = screenMenu
	applied := time.Date(2026, 11, 6, 22, 0, 0, 0, time.Local)
	m.data = status.Data{
		Image: status.Image{
			Booted:  status.Deployment{Version: "41.20251101.0"},
			Prev:    &status.Deployment{Version: "41.20251020.0"},
			Applied: &applied,
			Changes: []status.Change{
				{Kind: status.KindUp, Name: "bash", From: "5.2.37-1.fc41", To: "5.2.38-1.fc41"},
			},
		},
		Flatpak: status.Flatpak{Days: 30, Versions: map[string]string{}},
		Timers:  []status.Timer{},
	}
	m.setDetailContent()
	return m
}

func TestMenuView(t *testing.T) {
	m := testModel(t)
	view := m.View()
	if !strings.Contains(view, "UPDATE STATUS") {
		t.Fatal("menu is missing its title")
	}
	for _, want := range []string{"Image", "Flatpak", "AppImage", "Auto-updates", "Everything"} {
		if !strings.Contains(view, want) {
			t.Errorf("menu missing %q", want)
		}
	}
	if n := strings.Count(view, "▌"); n != 1 {
		t.Errorf("menu has %d selection bars; want exactly 1", n)
	}
	lines := strings.Split(view, "\n")
	if len(lines) != m.height-1 {
		t.Errorf("frame has %d lines; want %d", len(lines), m.height-1)
	}
	wantWidth := m.frameWidth()
	for _, l := range lines {
		if got := ansi.StringWidth(l); got != wantWidth {
			t.Errorf("line width %d; want %d: %q", got, wantWidth, l)
		}
	}
}

func TestSelectionSpringGlides(t *testing.T) {
	m := testModel(t)
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = tm.(model)
	if m.cursor != 1 {
		t.Fatalf("cursor = %d; want 1", m.cursor)
	}
	for i := range 400 {
		tm, _ = m.Update(animTick{})
		m = tm.(model)
		if m.selY == 1 && !m.animating {
			break
		}
		if i == 399 {
			t.Fatalf("spring did not settle: selY=%v selV=%v", m.selY, m.selV)
		}
	}
	if m.selY != 1 {
		t.Fatalf("selY = %v; want 1", m.selY)
	}
	view := m.View()
	if n := strings.Count(view, "▌"); n != 1 {
		t.Fatalf("%d selection bars after settling", n)
	}
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "▣ Flatpak") && !strings.Contains(line, "▌") {
			t.Errorf("highlight is not on the Flatpak row: %q", line)
		}
	}
}

func TestOpenDetailAndScroll(t *testing.T) {
	m := testModel(t)
	m.opts.All = true
	m.data.Image.Changes = nil
	for i := range 40 {
		m.data.Image.Changes = append(m.data.Image.Changes, status.Change{
			Kind: status.KindUp, Name: "pkg" + strings.Repeat("x", i%5), From: "1", To: "2",
		})
	}
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = tm.(model)
	if m.screen != screenDetail || m.viewing != 0 {
		t.Fatalf("screen=%v viewing=%d", m.screen, m.viewing)
	}
	if !strings.Contains(m.vp.View(), "IMAGE") {
		t.Error("detail viewport has no image card")
	}
	if lipgloss.Height(m.vp.View()) != m.bodyHeight() {
		t.Errorf("viewport height = %d; want %d", lipgloss.Height(m.vp.View()), m.bodyHeight())
	}
	view := m.View()
	if !strings.Contains(view, "╭─ ◆ IMAGE") {
		t.Error("detail view is missing the image card border")
	}
	// Detail frame is exactly terminal height minus one.
	if got := lipgloss.Height(view); got != m.height-1 {
		t.Errorf("detail frame height = %d; want %d", got, m.height-1)
	}
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = tm.(model)
	if m.vp.YOffset != 1 {
		t.Errorf("YOffset = %d after one line down", m.vp.YOffset)
	}
}

func TestEverythingDetail(t *testing.T) {
	m := testModel(t)
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = tm.(model)
	if m.viewing != everything {
		t.Fatalf("viewing = %d", m.viewing)
	}
	content := m.vp.View()
	for _, want := range []string{"IMAGE", "FLATPAK", "APPIMAGE", "AUTO-UPDATES"} {
		if !strings.Contains(content, want) {
			t.Errorf("everything view missing %q", want)
		}
	}
}

func TestHelpScreenAndBack(t *testing.T) {
	m := testModel(t)
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = tm.(model)
	if m.screen != screenHelp {
		t.Fatalf("screen = %v", m.screen)
	}
	if !strings.Contains(m.View(), "Check for available updates") {
		t.Error("help screen content missing")
	}
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(model)
	if m.screen != screenMenu {
		t.Fatalf("esc did not return to the menu: %v", m.screen)
	}
}

func TestLoadTransitionsToMenu(t *testing.T) {
	m := newModel(Config{Options: status.DefaultOptions(), Version: "test"})
	m.width, m.height = 80, 24
	if m.screen != screenLoading {
		t.Fatalf("initial screen = %v", m.screen)
	}
	tm, _ := m.Update(loadedMsg{all: true, data: status.Data{}})
	m = tm.(model)
	if m.screen != screenMenu {
		t.Fatalf("screen after load = %v; want menu", m.screen)
	}
	if !m.loaded {
		t.Error("loaded flag not set")
	}
}

func TestResizeKeepsViewportInSync(t *testing.T) {
	m := testModel(t)
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = tm.(model)
	if m.vp.Width != m.bodyWidth() || m.vp.Height != m.bodyHeight() {
		t.Errorf("viewport = %dx%d; want %dx%d", m.vp.Width, m.vp.Height, m.bodyWidth(), m.bodyHeight())
	}
	if got := lipgloss.Height(m.View()); got != m.height-1 {
		t.Errorf("frame height = %d; want %d", got, m.height-1)
	}
}
