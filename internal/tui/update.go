package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"tustat/internal/status"
)

// animTick advances the harmonica spring that glides the selection bar.
type animTick struct{}

func animTickCmd() tea.Cmd {
	return tea.Tick(time.Second/60, func(time.Time) tea.Msg { return animTick{} })
}

// loadedMsg carries one finished gather to the model.
type loadedMsg struct {
	section status.Section
	all     bool
	check   bool
	data    status.Data
}

func (m model) loadAll(check bool) tea.Cmd {
	opts := m.opts
	opts.Check = opts.Check || check
	return func() tea.Msg { return loadedMsg{all: true, check: check, data: status.GatherAll(opts)} }
}

func (m model) loadSection(s status.Section, check bool) tea.Cmd {
	opts := m.opts
	opts.Check = check
	return func() tea.Msg {
		return loadedMsg{section: s, check: check, data: status.GatherSelected(opts, []status.Section{s})}
	}
}

// Update handles all messages.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.setDetailContent()
		return m, nil

	case loadedMsg:
		m.loaded = true
		m.activity = ""
		m.opts.Now = time.Now()
		if msg.all {
			m.data = msg.data
		} else {
			m.data = m.data.Merge(msg.section, msg.data)
		}
		if m.screen == screenLoading {
			m.screen = screenMenu
		}
		m.setDetailContent()
		return m, nil

	case animTick:
		if m.screen != screenMenu {
			m.animating = false
			return m, nil
		}
		m.selY, m.selV = m.spring.Update(m.selY, m.selV, float64(m.cursor))
		if near(m.selY, float64(m.cursor)) && near(m.selV, 0) {
			m.selY, m.selV = float64(m.cursor), 0
			m.animating = false
			return m, nil
		}
		return m, animTickCmd()

	case spinner.TickMsg:
		if m.activity == "" && m.loaded {
			m.spinRunning = false
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func near(a, b float64) bool {
	d := a - b
	return d < 0.01 && d > -0.01
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	switch m.screen {
	case screenLoading:
		return m, nil
	case screenMenu:
		return m.menuKey(key)
	case screenDetail:
		return m.detailKey(key)
	case screenHelp:
		switch key {
		case "esc", "backspace", "?":
			m.screen = m.prev
			if m.screen == screenHelp {
				m.screen = screenMenu
			}
		}
		return m, nil
	}
	return m, nil
}

func (m model) menuKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		return m.moveCursor(-1)
	case "down", "j":
		return m.moveCursor(1)
	case "home", "g":
		m.cursor = 0
		return m.startAnim()
	case "end", "G":
		m.cursor = menuRows - 1
		return m.startAnim()
	case "r":
		var cmd tea.Cmd
		m, cmd = m.beginActivity("refreshing…")
		return m, tea.Batch(m.loadAll(false), cmd)
	case "?":
		m.prev = screenMenu
		m.screen = screenHelp
	case "enter":
		return m.openDetail(m.cursor)
	case "1", "2", "3", "4", "5":
		return m.openDetail(int(key[0] - '1'))
	}
	return m, nil
}

func (m model) moveCursor(delta int) (tea.Model, tea.Cmd) {
	cursor := m.cursor + delta
	if cursor < 0 || cursor >= menuRows {
		return m, nil
	}
	m.cursor = cursor
	return m.startAnim()
}

// startAnim kicks the spring loop if it is not already running.
func (m model) startAnim() (model, tea.Cmd) {
	if m.animating {
		return m, nil
	}
	m.animating = true
	return m, animTickCmd()
}

func (m model) openDetail(index int) (tea.Model, tea.Cmd) {
	m.viewing = index
	m.screen = screenDetail
	m.setDetailContent()
	m.vp.GotoTop()
	return m, nil
}

// beginActivity shows the spinner and returns its first tick when needed.
func (m model) beginActivity(text string) (model, tea.Cmd) {
	m.activity = text
	if m.spinRunning {
		return m, nil
	}
	m.spinRunning = true
	return m, m.spin.Tick
}

func (m model) detailKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "backspace", "h":
		m.screen = screenMenu
		return m, nil
	case "up", "k":
		m.vp.LineUp(1)
	case "down", "j":
		m.vp.LineDown(1)
	case "pgup", "b":
		m.vp.HalfPageUp()
	case "pgdown", " ":
		m.vp.HalfPageDown()
	case "home", "g":
		m.vp.GotoTop()
	case "end", "G":
		m.vp.GotoBottom()
	case "?":
		m.prev = screenDetail
		m.screen = screenHelp
	case "r":
		var cmd tea.Cmd
		m, cmd = m.beginActivity("refreshing…")
		return m, tea.Batch(m.reload(false), cmd)
	case "c":
		var cmd tea.Cmd
		m, cmd = m.beginActivity("checking for available updates…")
		return m, tea.Batch(m.reload(true), cmd)
	}
	return m, nil
}

func (m model) reload(check bool) tea.Cmd {
	if m.viewing == everything {
		return m.loadAll(check)
	}
	return m.loadSection(status.Section(m.viewing), check)
}
