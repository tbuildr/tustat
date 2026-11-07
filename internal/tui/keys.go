package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap is the single source of truth for bindings and the footer help.
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Select   key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Refresh  key.Binding
	Check    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		PageUp:   key.NewBinding(key.WithKeys("pgup", "b"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown", " "), key.WithHelp("pgdn", "page down")),
		Top:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:     key.NewBinding(key.WithKeys("esc", "backspace", "h"), key.WithHelp("esc", "back")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Check:    key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "check")),
	}
}

// short returns the footer bindings for the current screen.
func (k keyMap) short(s screen) []key.Binding {
	switch s {
	case screenMenu:
		return []key.Binding{k.Up, k.Down, k.Select, k.Refresh, k.Help, k.Quit}
	case screenDetail:
		return []key.Binding{k.Up, k.Down, k.Check, k.Refresh, k.Back, k.Quit}
	case screenHelp:
		return []key.Binding{k.Back, k.Quit}
	default:
		return []key.Binding{k.Quit}
	}
}

func (k keyMap) full(s screen) [][]key.Binding {
	switch s {
	case screenMenu:
		return [][]key.Binding{{k.Up, k.Down, k.Select}, {k.Refresh, k.Help, k.Quit}}
	case screenDetail:
		return [][]key.Binding{
			{k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom},
			{k.Check, k.Refresh, k.Back, k.Quit},
		}
	default:
		return [][]key.Binding{{k.Back, k.Quit}}
	}
}

// contextualKeys adapts the key map to the bubbles help component.
type contextualKeys struct {
	keys   keyMap
	screen screen
}

func (c contextualKeys) ShortHelp() []key.Binding  { return c.keys.short(c.screen) }
func (c contextualKeys) FullHelp() [][]key.Binding { return c.keys.full(c.screen) }
