package tui

import "github.com/charmbracelet/lipgloss"

// Palette: restrained ANSI colours that follow the terminal theme, with
// the same blue accent the rest of the tbzos tooling uses.
var (
	colAccent = lipgloss.Color("4")
	colOK     = lipgloss.Color("2")
	colWarn   = lipgloss.Color("3")
	colDanger = lipgloss.Color("1")
	colDim    = lipgloss.Color("8")
	colText   = lipgloss.Color("15")
	colBar    = lipgloss.Color("6")
	colMagic  = lipgloss.Color("5")
)

var (
	styleFrame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAccent).
			Padding(0, 2)

	styleBrand = lipgloss.NewStyle().
			Bold(true).
			Foreground(colMagic)

	styleVersion = lipgloss.NewStyle().
			Foreground(colDim)

	styleContext = lipgloss.NewStyle().
			Foreground(colDim)

	styleActivity = lipgloss.NewStyle().
			Foreground(colWarn)

	styleDivider = lipgloss.NewStyle().
			Foreground(colDim)

	stylePanelTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent)

	styleItem = lipgloss.NewStyle().
			Foreground(colText)

	styleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(colText)

	styleBar = lipgloss.NewStyle().
			Bold(true).
			Foreground(colBar)

	styleDesc = lipgloss.NewStyle().
			Foreground(colDim)

	styleFooter = lipgloss.NewStyle().
			Foreground(colDim)

	styleKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent)

	styleErr = lipgloss.NewStyle().
			Bold(true).
			Foreground(colDanger)

	styleWarn = lipgloss.NewStyle().
			Bold(true).
			Foreground(colWarn)

	styleOK = lipgloss.NewStyle().
		Bold(true).
		Foreground(colOK)

	styleMagic = lipgloss.NewStyle().
			Foreground(colMagic)

	styleHelpTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colText)
)
