package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"tustat/internal/render"
	"tustat/internal/status"
)

// View renders the current screen inside the framed shell.
func (m model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	width := m.bodyWidth()
	frameW := m.frameWidth()
	contentH := maxInt(m.height-3, 10)
	bodyH := m.bodyHeight()

	header := m.viewHeader(width)
	m.help.Width = width
	footer := render.Truncate(m.help.View(contextualKeys{keys: m.keys, screen: m.screen}), width)

	var body string
	switch m.screen {
	case screenDetail:
		body = m.vp.View()
	case screenHelp:
		body = m.viewHelp(width)
	case screenMenu:
		if m.loaded {
			body = m.viewMenu(width)
		} else {
			body = m.viewLoading(width)
		}
	default:
		body = m.viewLoading(width)
	}
	body = fitHeight(body, bodyH)

	content := lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer)
	return styleFrame.
		Width(frameW - 2).
		Height(contentH).
		MaxWidth(frameW).
		MaxHeight(contentH + 2).
		Render(content)
}

func (m model) viewHeader(width int) string {
	brand := styleBrand.Render("⚡ tustat") + " " + styleVersion.Render("v"+m.version)
	right := styleContext.Render(m.contextLabel())
	if m.activity != "" {
		right = m.spin.View() + " " + styleActivity.Render(m.activity) + "   " + right
	}
	gap := width - lipgloss.Width(brand) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := brand + strings.Repeat(" ", gap) + render.Truncate(right, maxInt(8, width-lipgloss.Width(brand)-gap))
	divider := styleDivider.Render(strings.Repeat("─", width))
	return line + "\n" + divider
}

func (m model) viewMenu(width int) string {
	var b strings.Builder
	b.WriteString(stylePanelTitle.Render("UPDATE STATUS") + "\n")
	b.WriteString(styleDivider.Render(strings.Repeat("─", minInt(width, 44))) + "\n")

	// Only the row the spring is currently over is highlighted, so the bar
	// glides between rows instead of jumping.
	selectedRow := int(m.selY + 0.5)
	v := render.NewView(m.theme, m.opts.Now, m.opts.Home, m.opts.All)
	for i := range menuRows {
		if i < everything {
			s := status.Section(i)
			b.WriteString(v.SummaryRow(s, status.Summarize(s, m.data), i+1, i == selectedRow, width))
		} else {
			b.WriteString(m.everythingRow(i == selectedRow, width))
		}
		b.WriteString("\n")
	}
	if width >= 56 {
		legend := styleOK.Render("●") + styleDesc.Render(" under a week   ") +
			styleWarn.Render("●") + styleDesc.Render(" under a month   ") +
			styleErr.Render("●") + styleDesc.Render(" older")
		if !m.opts.Now.IsZero() && lipgloss.Width(legend)+30 <= width {
			legend += styleDesc.Render("          refreshed " + m.opts.Now.Format("15:04"))
		}
		b.WriteString("\n  " + legend + "\n")
	}
	return b.String()
}

func (m model) everythingRow(selected bool, width int) string {
	bar := " "
	name := "Everything"
	if selected {
		bar = styleBar.Render("▌")
		name = styleSelected.Render(name)
	} else {
		name = styleItem.Render(name)
	}
	head := bar + styleDesc.Render("5") + "  " + styleMagic.Render("≡") + " " + render.Pad(name, 14) + " "
	desc := "all sections on one page"
	room := width - lipgloss.Width(head)
	if room < 8 {
		room = 8
	}
	return head + styleDesc.Render(render.Truncate(desc, room))
}

func (m model) viewLoading(width int) string {
	msg := m.activity
	if msg == "" {
		msg = "gathering update status…"
	}
	var b strings.Builder
	b.WriteString(stylePanelTitle.Render("UPDATE STATUS") + "\n")
	b.WriteString(styleDivider.Render(strings.Repeat("─", minInt(width, 44))) + "\n")
	b.WriteString("\n" + m.spin.View() + " " + styleActivity.Render(msg) + "\n\n")
	b.WriteString(styleDesc.Render("  reading rpm-ostree, Flatpak, AppImages and update timers") + "\n")
	return b.String()
}

func (m model) viewHelp(width int) string {
	var b strings.Builder
	b.WriteString(styleHelpTitle.Render("Help") + "\n")
	b.WriteString(styleDivider.Render(strings.Repeat("─", minInt(width, 44))) + "\n")
	rows := [][2]string{
		{"↑ / k", "Move up"},
		{"↓ / j", "Move down / scroll"},
		{"Enter", "Open a section"},
		{"1–5", "Jump straight to a section"},
		{"Esc / h", "Back to the menu"},
		{"PgUp/PgDn", "Page the detail view"},
		{"g / G", "Top / bottom"},
		{"r", "Refresh what is shown"},
		{"c", "Check for available updates"},
		{"?", "This help"},
		{"q", "Quit"},
	}
	for _, r := range rows {
		b.WriteString("  " + styleKey.Render(render.Pad(r[0], 10)) + styleItem.Render(r[1]) + "\n")
	}
	b.WriteString("\n" + styleDesc.Render("  tustat is read-only: it only writes the AppImage snapshot and change log.") + "\n")
	return b.String()
}

// fitHeight clips or pads a rendered block to exactly h lines.
func fitHeight(s string, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
