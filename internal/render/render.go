// Package render paints tustat's status cards and summary rows. The same
// lines feed both the plain print mode and the Bubble Tea detail views.
package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"tustat/internal/status"
	"tustat/internal/timefmt"
)

// Theme holds the styled brushes used to paint status text.
type Theme struct {
	R       *lipgloss.Renderer
	Text    lipgloss.Style
	Dim     lipgloss.Style
	Bold    lipgloss.Style
	Accent  lipgloss.Style
	Magenta lipgloss.Style
	Cyan    lipgloss.Style
	Green   lipgloss.Style
	Yellow  lipgloss.Style
	Red     lipgloss.Style
	Bar     lipgloss.Style
	Border  lipgloss.Style
}

// NewTheme builds a theme from a renderer (so print mode and the TUI can
// have independent colour profiles).
func NewTheme(r *lipgloss.Renderer) *Theme {
	if r == nil {
		r = lipgloss.DefaultRenderer()
	}
	return &Theme{
		R:       r,
		Text:    r.NewStyle(),
		Dim:     r.NewStyle().Foreground(lipgloss.Color("8")),
		Bold:    r.NewStyle().Bold(true),
		Accent:  r.NewStyle().Foreground(lipgloss.Color("4")),
		Magenta: r.NewStyle().Foreground(lipgloss.Color("5")),
		Cyan:    r.NewStyle().Foreground(lipgloss.Color("6")),
		Green:   r.NewStyle().Foreground(lipgloss.Color("2")),
		Yellow:  r.NewStyle().Foreground(lipgloss.Color("3")),
		Red:     r.NewStyle().Foreground(lipgloss.Color("1")),
		Bar:     r.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
		Border:  r.NewStyle().Foreground(lipgloss.Color("8")),
	}
}

// Tone paints s in the given semantic tone.
func (t *Theme) Tone(tone status.Tone, s string) string {
	switch tone {
	case status.ToneOK:
		return t.Green.Render(s)
	case status.ToneWarn:
		return t.Yellow.Render(s)
	case status.ToneDanger:
		return t.Red.Render(s)
	case status.ToneDim:
		return t.Dim.Render(s)
	default:
		return t.Text.Render(s)
	}
}

// Tier paints s according to an age tier.
func (t *Theme) Tier(tier timefmt.Tier, s string) string {
	switch tier {
	case timefmt.TierFresh:
		return t.Green.Render(s)
	case timefmt.TierAgeing:
		return t.Yellow.Render(s)
	default:
		return t.Red.Render(s)
	}
}

// Marker paints a change marker.
func (t *Theme) Marker(k status.Kind) string {
	switch k {
	case status.KindUp:
		return t.Cyan.Render(k.Marker())
	case status.KindDown:
		return t.Magenta.Render(k.Marker())
	case status.KindAdd:
		return t.Green.Render(k.Marker())
	default:
		return t.Red.Render(k.Marker())
	}
}

// Truncate shortens s to n visible cells, appending an ellipsis.
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= n {
		return s
	}
	return ansi.Truncate(s, n, "…")
}

// Pad right-pads (or truncates) s to exactly n cells.
func Pad(s string, n int) string {
	if n <= 0 {
		return ""
	}
	s = Truncate(s, n)
	return s + strings.Repeat(" ", n-ansi.StringWidth(s))
}

// View renders status data at a fixed width.
type View struct {
	Theme *Theme
	Now   time.Time
	Home  string
	All   bool
}

// NewView builds a view.
func NewView(t *Theme, now time.Time, home string, all bool) *View {
	return &View{Theme: t, Now: now, Home: home, All: all}
}

// Brand is the product name used in headers.
const Brand = "⚡ tustat"

func (v *View) brandLine() string {
	return v.Theme.Magenta.Render(v.Theme.Bold.Render(Brand))
}

// HeaderLine is the print-mode brand line.
func (v *View) HeaderLine(host, version string) string {
	parts := []string{
		v.brandLine(),
		v.Theme.Dim.Render(version),
		v.Theme.Dim.Render("·"),
		v.Theme.Text.Render(host),
		v.Theme.Dim.Render("·"),
		v.Theme.Dim.Render(timefmt.When(v.Now, v.Now)),
	}
	return "  " + strings.Join(parts, " ")
}

// Dot is the age-coloured status dot, grey when there is no timestamp.
func (v *View) Dot(t *time.Time) string {
	if t == nil {
		return v.Theme.Dim.Render("●")
	}
	return v.Theme.Tier(timefmt.AgeTier(v.Now, *t), "●")
}

// AgeText renders "2h ago" or an em dash.
func (v *View) AgeText(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return timefmt.Ago(v.Now, *t)
}

// SummaryRow renders one menu/print row for a section. Narrow terminals
// give up cells from the age and name columns before the detail does.
func (v *View) SummaryRow(s status.Section, sum status.Summary, index int, selected bool, width int) string {
	t := v.Theme
	bar := " "
	if selected {
		bar = t.Bar.Render("▌")
	}
	num := " "
	if index > 0 {
		num = fmt.Sprintf("%d", index)
	}
	headFor := func(nameW int) string {
		name := s.Name()
		if selected {
			name = t.Bold.Render(name)
		} else {
			name = t.Text.Render(name)
		}
		return bar + t.Dim.Render(num) + "  " + t.Magenta.Render(s.Icon()) + " " + Pad(name, nameW) + " "
	}

	nameW, ageW := 14, 12
	head := headFor(nameW)
	fixed := ansi.StringWidth(head) + 1 + 1 + ageW + 1
	for fixed+6 > width && ageW > 8 {
		ageW--
		fixed--
	}
	for fixed+6 > width && nameW > 8 {
		nameW--
		head = headFor(nameW)
		fixed--
	}
	head += v.Dot(sum.T) + " " + Pad(v.AgeText(sum.T), ageW) + " "
	detailWidth := width - ansi.StringWidth(head)
	if detailWidth < 1 {
		detailWidth = 1
	}
	row := head + t.Tone(sum.Tone, Truncate(sum.Detail, detailWidth))
	return Truncate(row, width)
}

// Section renders one section as a bordered card.
func (v *View) Section(s status.Section, d status.Data, width int) []string {
	if width < 28 {
		width = 28
	}
	switch s {
	case status.SectionImage:
		return v.imageCard(d.Image, width)
	case status.SectionFlatpak:
		return v.flatpakCard(d.Flatpak, width)
	case status.SectionAppImage:
		return v.appImageCard(d.AppImages, width)
	default:
		return v.timerCard(d.Timers, width)
	}
}

// Everything renders every section, separated by a blank line.
func (v *View) Everything(data status.Data, width int) []string {
	var out []string
	for i, s := range status.Sections() {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, v.Section(s, data, width)...)
	}
	return out
}

type card struct {
	icon  string
	title string
	right string
	tone  status.Tone
	lines []string
}

func (v *View) renderCard(c card, width int) []string {
	t := v.Theme
	inner := width - 4
	if inner < 10 {
		inner = 10
	}
	left := "╭─ " + t.Magenta.Render(c.icon) + " " + t.Bold.Render(c.title) + " "
	right := ""
	if c.right != "" {
		right = t.Tone(c.tone, c.right)
	}
	// Drop the right label when it cannot fit beside the title.
	if ansi.StringWidth(left)+ansi.StringWidth(right)+5 > width {
		right = ""
	}
	left = Truncate(left, width-2)
	rem := width - ansi.StringWidth(left) - 1
	if right != "" {
		rem -= ansi.StringWidth(right) + 2
	}
	if rem < 1 {
		rem = 1
	}
	top := left + t.Border.Render(strings.Repeat("─", rem))
	if right != "" {
		top += " " + right + " "
	}
	top += t.Border.Render("╮")

	out := []string{top}
	for _, line := range c.lines {
		line = Truncate(line, inner)
		out = append(out, t.Border.Render("│")+" "+line+strings.Repeat(" ", inner-ansi.StringWidth(line))+" "+t.Border.Render("│"))
	}
	out = append(out, t.Border.Render("╰"+strings.Repeat("─", width-2)+"╯"))
	return out
}

func (v *View) kv(label, value string) string {
	return "  " + v.Theme.Dim.Render(Pad(label, 11)) + value
}

func (v *View) heading(text, extra string) string {
	out := "  " + v.Theme.Bold.Render(text)
	if extra != "" {
		out += "  " + v.Theme.Dim.Render(extra)
	}
	return out
}

func (v *View) note(tone status.Tone, text string) string {
	return "  " + v.Theme.Tone(tone, text)
}

func (v *View) imageCard(im status.Image, width int) []string {
	t := v.Theme
	if im.Err != "" {
		return v.renderCard(card{
			icon: "◆", title: "IMAGE", lines: []string{v.note(status.ToneWarn, "! "+im.Err)},
		}, width)
	}

	right, tone := "applied ?", status.ToneDim
	if im.Applied != nil {
		right = "applied " + timefmt.Ago(v.Now, *im.Applied)
		tone = toneForTier(timefmt.AgeTier(v.Now, *im.Applied))
	}

	var lines []string
	b := im.Booted
	lines = append(lines, v.kv("Booted", t.Bold.Render(b.Version)))

	signed := strings.HasPrefix(b.Ref, "ostree-image-signed")
	ref := b.Ref
	if i := strings.Index(ref, "://"); i >= 0 {
		ref = ref[i+3:]
	}
	if signed {
		lines = append(lines, v.kv("Image", ref+"  "+t.Green.Render("✓ signed")))
	} else {
		lines = append(lines, v.kv("Image", ref))
	}
	if b.Digest != "" {
		lines = append(lines, v.kv("Digest", t.Dim.Render(status.ShortDigest(b.Digest))))
	}
	if !b.Built.IsZero() {
		lines = append(lines, v.kv("Built", timefmt.When(v.Now, b.Built)+"  "+t.Dim.Render(timefmt.Ago(v.Now, b.Built))))
	}
	if im.Applied != nil {
		applied := *im.Applied
		lines = append(lines, v.kv("Applied",
			v.Theme.Tier(timefmt.AgeTier(v.Now, applied), timefmt.When(v.Now, applied))+"  "+
				t.Dim.Render(timefmt.Ago(v.Now, applied))+" "+t.Dim.Render("· finalized at reboot")))
	} else {
		lines = append(lines, v.kv("Applied", t.Yellow.Render("unknown — journal rotated or unreadable")))
	}
	if im.Staged != nil {
		lines = append(lines, v.kv("Staged",
			t.Yellow.Render("⟳ "+im.Staged.Version)+"  "+t.Dim.Render("reboot to apply")))
		if im.StagedAt != nil {
			lines = append(lines, v.kv("Staged at",
				timefmt.When(v.Now, *im.StagedAt)+"  "+t.Dim.Render(timefmt.Ago(v.Now, *im.StagedAt))))
		}
	} else {
		lines = append(lines, v.kv("Staged", t.Green.Render("✓ nothing pending")))
	}

	if im.Prev != nil {
		lines = append(lines, "")
		lines = append(lines, v.heading("What was applied", fmt.Sprintf("%s → %s", im.Prev.Version, b.Version)))
		lines = append(lines, v.changes(im.Changes, im.DiffErr, width)...)
	} else {
		lines = append(lines, "")
		lines = append(lines, v.note(status.ToneDim, "no previous deployment to compare against"))
	}
	if im.Staged != nil {
		lines = append(lines, "")
		lines = append(lines, v.heading("Pending after reboot", fmt.Sprintf("%s → %s", b.Version, im.Staged.Version)))
		lines = append(lines, v.changes(im.StagedChanges, "", width)...)
	}
	if im.Check != nil {
		lines = append(lines, "")
		lines = append(lines, v.heading("Available", "rpm-ostree upgrade --check"))
		for _, l := range strings.Split(*im.Check, "\n") {
			lines = append(lines, "  "+expandTabs(l))
		}
	}
	return v.renderCard(card{icon: "◆", title: "IMAGE", right: right, tone: tone, lines: lines}, width)
}

const maxItems = 20

func (v *View) changes(list []status.Change, errText string, width int) []string {
	t := v.Theme
	if len(list) == 0 {
		if errText != "" {
			return []string{v.note(status.ToneWarn, "! "+firstLine(errText))}
		}
		return []string{v.note(status.ToneDim, "no package changes")}
	}

	counts := [4]int{}
	for _, c := range list {
		counts[c.Kind]++
	}
	var parts []string
	for _, k := range []status.Kind{status.KindUp, status.KindDown, status.KindAdd, status.KindRemove} {
		if counts[k] == 0 {
			continue
		}
		parts = append(parts, t.Marker(k)+" "+t.Bold.Render(fmt.Sprint(counts[k]))+" "+t.Text.Render(k.Label()))
	}
	out := []string{"  " + strings.Join(parts, t.Dim.Render("  ·  "))}

	nameW := 0
	for _, c := range list {
		if w := ansi.StringWidth(c.Name); w > nameW {
			nameW = w
		}
	}
	if nameW > 30 {
		nameW = 30
	}
	const indent = 2
	maxW := width - 6
	avail := maxW - (indent + 2 + nameW + 2)
	if avail < 16 {
		avail = 16
	}
	half := avail / 2

	limit := len(list)
	if !v.All && limit > maxItems {
		limit = maxItems
	}
	for _, c := range list[:limit] {
		row := "  " + t.Marker(c.Kind) + " " + Pad(c.Name, nameW) + "  "
		switch {
		case c.From != "":
			row += t.Dim.Render(Truncate(c.From, half)) + " " + t.Dim.Render("→") + " " + Truncate(c.To, half)
		case c.Kind == status.KindRemove:
			row += t.Dim.Render(Truncate(c.To, avail))
		default:
			row += Truncate(c.To, avail)
		}
		out = append(out, row)
	}
	if limit < len(list) {
		out = append(out, v.note(status.ToneDim, fmt.Sprintf("    … %d more  (tustat -a shows all)", len(list)-limit)))
	}
	return out
}

func (v *View) flatpakCard(fp status.Flatpak, width int) []string {
	t := v.Theme
	if fp.Err != "" {
		return v.renderCard(card{
			icon: "▣", title: "FLATPAK", lines: []string{v.note(status.ToneWarn, "! "+fp.Err)},
		}, width)
	}
	if len(fp.Entries) == 0 {
		return v.renderCard(card{
			icon: "▣", title: "FLATPAK", right: "nothing recent", tone: status.ToneDim,
			lines: []string{v.note(status.ToneDim, fmt.Sprintf("no changes in the last %d days (or the system journal isn't readable)", fp.Days))},
		}, width)
	}

	last := fp.Entries[len(fp.Entries)-1]
	right, tone := "applied", status.ToneDim
	if !last.T.IsZero() {
		right = "applied " + timefmt.Ago(v.Now, last.T)
		tone = toneForTier(timefmt.AgeTier(v.Now, last.T))
	}

	var lines []string
	if !last.T.IsZero() {
		lines = append(lines, v.kv("Last run",
			t.Tier(timefmt.AgeTier(v.Now, last.T), timefmt.When(v.Now, last.T))+"  "+
				t.Dim.Render(timefmt.Ago(v.Now, last.T))))
	} else {
		lines = append(lines, v.kv("Last run", last.Day))
	}

	nameW, count := 0, 0
	for _, e := range fp.Entries {
		if e.Day != last.Day {
			continue
		}
		count++
		if w := ansi.StringWidth(e.App); w > nameW {
			nameW = w
		}
	}
	if nameW > 40 {
		nameW = 40
	}
	label := last.Day
	if !last.T.IsZero() {
		label = timefmt.DayLabel(last.T)
	}
	lines = append(lines, "")
	lines = append(lines, v.heading("What was applied", fmt.Sprintf("%s · %d change%s", label, count, plural(count))))
	for _, e := range fp.Entries {
		if e.Day != last.Day {
			continue
		}
		ver := fp.Versions[e.App]
		lines = append(lines, "  "+t.Marker(e.Kind)+" "+Pad(e.App, nameW)+"  "+
			t.Dim.Render(Pad(e.Branch, 8)+Pad(e.Inst, 8))+Pad(ver, 14)+t.Dim.Render(e.Clock))
	}

	// Earlier days: day → count, newest first.
	type dayGroup struct {
		day   string
		stamp time.Time
		count int
	}
	var days []dayGroup
	for _, e := range fp.Entries {
		if e.Day == last.Day {
			continue
		}
		if len(days) > 0 && days[len(days)-1].day == e.Day {
			days[len(days)-1].count++
			continue
		}
		days = append(days, dayGroup{day: e.Day, stamp: e.T, count: 1})
	}
	lines = append(lines, "")
	lines = append(lines, v.heading("Earlier", fmt.Sprintf("last %d days", fp.Days)))
	if len(days) == 0 {
		lines = append(lines, v.note(status.ToneDim, "  none"))
	}
	shown := 0
	for i := len(days) - 1; i >= 0; i-- {
		if !v.All && shown == 8 {
			lines = append(lines, v.note(status.ToneDim, fmt.Sprintf("  … %d more days  (tustat -a)", i+1)))
			break
		}
		shown++
		g := days[i]
		lbl := g.day
		if !g.stamp.IsZero() {
			lbl = timefmt.DayLabel(g.stamp)
		}
		bars := strings.Repeat("▪", min(g.count, 24))
		lines = append(lines, "  "+Pad(lbl, 12)+t.Cyan.Render(bars)+" "+t.Dim.Render(fmt.Sprint(g.count)))
	}

	if fp.Check != nil {
		lines = append(lines, "")
		lines = append(lines, v.heading("Available", "flatpak remote-ls --updates"))
		if *fp.Check == "" {
			lines = append(lines, v.note(status.ToneOK, "  ✓ everything up to date"))
		}
		for _, l := range strings.Split(*fp.Check, "\n") {
			if l == "" {
				continue
			}
			f := strings.Split(l, "\t")
			app := f[0]
			ver := ""
			if len(f) > 1 {
				ver = f[1]
			}
			lines = append(lines, "  "+t.Marker(status.KindUp)+" "+Pad(app, 40)+"  "+ver)
		}
	}
	return v.renderCard(card{icon: "▣", title: "FLATPAK", right: right, tone: tone, lines: lines}, width)
}

func (v *View) appImageCard(ai status.AppImages, width int) []string {
	t := v.Theme
	var lines []string
	c := card{icon: "◇", title: "APPIMAGE"}
	if len(ai.Items) == 0 {
		c.right, c.tone = "none found", status.ToneDim
		dirs := make([]string, 0, len(ai.Dirs))
		for _, d := range ai.Dirs {
			dirs = append(dirs, v.tilde(d))
		}
		lines = append(lines, v.note(status.ToneDim, "no AppImages in "+strings.Join(dirs, ", ")))
	} else {
		newest := ai.Items[0].MTime
		c.right = "applied " + timefmt.Ago(v.Now, newest)
		c.tone = toneForTier(timefmt.AgeTier(v.Now, newest))
		lines = append(lines, v.kv("Last run",
			t.Tier(timefmt.AgeTier(v.Now, newest), timefmt.When(v.Now, newest))+"  "+
				t.Dim.Render(timefmt.Ago(v.Now, newest))))
		lines = append(lines, v.kv("Tracked", fmt.Sprintf("%d app%s", len(ai.Items), plural(len(ai.Items)))))
		lines = append(lines, "")

		nameW, verW := 4, 7
		for _, a := range ai.Items {
			nameW = max(nameW, ansi.StringWidth(a.Name))
			verW = max(verW, ansi.StringWidth(a.Version))
		}
		nameW = min(nameW, 28)
		verW = min(verW, 16)
		lines = append(lines, "  "+t.Dim.Render(Pad("APP", nameW+2)+Pad("VERSION", verW+2)+"UPDATED"))
		for _, a := range ai.Items {
			lines = append(lines, "  "+t.Bold.Render(Pad(a.Name, nameW+2))+Pad(a.Version, verW+2)+
				t.Tier(timefmt.AgeTier(v.Now, a.MTime), timefmt.When(v.Now, a.MTime))+"  "+
				t.Dim.Render(timefmt.Ago(v.Now, a.MTime)))
		}
	}

	lines = append(lines, "")
	switch {
	case ai.FirstRun:
		lines = append(lines, v.note(status.ToneDim, "baseline recorded — updates are logged from now on"))
		lines = append(lines, v.note(status.ToneDim, v.tilde(ai.LogPath)))
	case len(ai.LogEvents) == 0:
		lines = append(lines, v.note(status.ToneDim, "no changes logged yet"))
	default:
		lines = append(lines, v.heading("Change log", v.tilde(ai.LogPath)))
		for i := len(ai.LogEvents) - 1; i >= 0; i-- {
			e := ai.LogEvents[i]
			kind := status.KindUp
			if e.Kind == "removed" {
				kind = status.KindRemove
			} else if e.Kind == "added" {
				kind = status.KindAdd
			}
			row := "  " + t.Marker(kind) + " " + t.Dim.Render(e.Stamp) + "  " + Pad(e.File, 34)
			if i := strings.Index(e.Detail, " -> "); i >= 0 {
				row += "  " + t.Dim.Render(e.Detail[:i]) + " " + t.Dim.Render("→") + " " + e.Detail[i+4:]
			} else {
				row += "  " + e.Detail
			}
			lines = append(lines, row)
		}
	}
	if ai.Check != nil {
		lines = append(lines, "")
		lines = append(lines, v.heading("Available", "Gear Lever"))
		if *ai.Check == "" {
			lines = append(lines, v.note(status.ToneOK, "  ✓ everything up to date"))
		}
		for _, l := range strings.Split(*ai.Check, "\n") {
			if l != "" {
				lines = append(lines, "  "+strings.Join(strings.Split(l, "\t"), "  "))
			}
		}
	}
	c.lines = lines
	return v.renderCard(c, width)
}

func (v *View) timerCard(timers []status.Timer, width int) []string {
	t := v.Theme
	c := card{icon: "◷", title: "AUTO-UPDATES"}
	if len(timers) == 0 {
		c.right, c.tone = "none found", status.ToneDim
		c.lines = []string{v.note(status.ToneDim, "no update timers installed")}
		return v.renderCard(c, width)
	}
	if failing := status.TimersFailing(timers); failing > 0 {
		c.right, c.tone = fmt.Sprintf("%d failing", failing), status.ToneDanger
	} else {
		c.right, c.tone = "all ok", status.ToneOK
	}

	var lines []string
	for _, tm := range timers {
		glyph := t.Dim.Render("○")
		if tm.Last != nil {
			if tm.Result == "success" {
				glyph = t.Green.Render("✓")
			} else {
				glyph = t.Red.Render("✗")
			}
		}
		name := tm.Name
		if tm.User {
			name += " (user)"
		}
		enabled := "?"
		if tm.Enabled != "" {
			enabled = tm.Enabled
		}
		enabledStyle := t.Dim
		if tm.Enabled != "enabled" {
			enabledStyle = t.Yellow
		}
		last := "never"
		if tm.Last != nil {
			last = timefmt.Ago(v.Now, *tm.Last)
		}
		next := "—"
		if tm.Next != nil {
			next = timefmt.Ago(v.Now, *tm.Next)
		}
		row := "  " + glyph + " " + t.Bold.Render(Pad(name, 34)) +
			enabledStyle.Render(Pad(enabled, 10)) +
			t.Dim.Render("last") + " " + Pad(last, 12) + " " +
			t.Dim.Render("next") + " " + next
		if tm.Last != nil && tm.Result != "success" {
			row += "  " + t.Red.Render(tm.Result)
		}
		lines = append(lines, row)
	}
	c.lines = lines
	return v.renderCard(c, width)
}

// tilde abbreviates a path under the user's home directory.
func (v *View) tilde(path string) string {
	if v.Home != "" && strings.HasPrefix(path, v.Home) {
		return "~" + path[len(v.Home):]
	}
	return path
}

func toneForTier(t timefmt.Tier) status.Tone {
	switch t {
	case timefmt.TierFresh:
		return status.ToneOK
	case timefmt.TierAgeing:
		return status.ToneWarn
	default:
		return status.ToneDanger
	}
}

// expandTabs replaces tabs with spaces so card borders stay aligned.
func expandTabs(s string) string {
	if !strings.ContainsRune(s, '\t') {
		return s
	}
	return strings.ReplaceAll(s, "\t", "    ")
}

func firstLine(s string) string {
	s = strings.TrimLeft(s, " \n\t")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
