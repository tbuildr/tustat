package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"

	"tustat/internal/render"
	"tustat/internal/status"
)

// Config parameterises the TUI.
type Config struct {
	Options status.Options
	Version string
}

type screen int

const (
	screenLoading screen = iota
	screenMenu
	screenDetail
	screenHelp
)

// everything is the menu row that shows all sections on one page.
const everything = 4

const menuRows = 5

type model struct {
	opts    status.Options
	version string
	theme   *render.Theme

	width, height int
	data          status.Data
	loaded        bool
	activity      string
	spinRunning   bool

	screen   screen
	prev     screen
	cursor   int
	viewing  int
	checking bool

	vp   viewport.Model
	spin spinner.Model
	help help.Model
	keys keyMap

	selY, selV float64
	spring     harmonica.Spring
	animating  bool
}

func newModel(cfg Config) model {
	sp := spinner.New(spinner.WithSpinner(spinner.Line))
	sp.Style = lipgloss.NewStyle().Foreground(colAccent)

	h := help.New()
	h.Styles.ShortKey = styleKey
	h.Styles.ShortDesc = styleFooter
	h.Styles.ShortSeparator = styleFooter
	h.Styles.FullKey = styleKey
	h.Styles.FullDesc = styleFooter
	h.Styles.FullSeparator = styleFooter
	h.Styles.Ellipsis = styleFooter

	vp := viewport.New(60, 10)
	vp.SetContent("")

	return model{
		opts:    cfg.Options,
		version: cfg.Version,
		theme:   render.NewTheme(lipgloss.DefaultRenderer()),
		screen:  screenLoading,
		vp:      vp,
		spin:    sp,
		help:    h,
		keys:    newKeyMap(),
		spring:  harmonica.NewSpring(harmonica.FPS(60), 220.0, 22.0),

		activity:    "gathering update status…",
		spinRunning: true,
	}
}

// Init starts the first gather, the spinner and the menu spring.
func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadAll(false), m.spin.Tick)
}

func (m model) frameWidth() int {
	return clampInt(m.width-2, 46, 112)
}

func (m model) bodyWidth() int {
	return m.frameWidth() - 6
}

func (m model) bodyHeight() int {
	return maxInt(maxInt(m.height-3, 10)-5, 3)
}

// detailLines builds the card lines for the current detail view.
func (m model) detailLines() []string {
	width := m.bodyWidth()
	v := render.NewView(m.theme, m.opts.Now, m.opts.Home, m.opts.All)
	if m.viewing == everything {
		return v.Everything(m.data, width)
	}
	return v.Section(status.Section(m.viewing), m.data, width)
}

// setDetailContent refreshes the viewport, keeping the scroll position
// where the viewport allows it.
func (m *model) setDetailContent() {
	if m.width <= 0 {
		return
	}
	m.vp.Width = m.bodyWidth()
	m.vp.Height = m.bodyHeight()
	m.vp.SetContent(strings.Join(m.detailLines(), "\n"))
}

func (m model) contextLabel() string {
	if !m.loaded {
		return "Starting"
	}
	switch m.screen {
	case screenMenu:
		if m.cursor == everything {
			return "Everything"
		}
		return status.Section(m.cursor).Name()
	case screenDetail:
		if m.viewing == everything {
			return "Everything"
		}
		return status.Section(m.viewing).Name()
	case screenHelp:
		return "Help"
	}
	return "Overview"
}

// startedAt is the time used for relative ages in the footer-free views.
func (m model) now() time.Time {
	if m.opts.Now.IsZero() {
		return time.Now()
	}
	return m.opts.Now
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
