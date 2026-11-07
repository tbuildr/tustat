// Package cli parses tustat's arguments and chooses between the plain
// print mode and the interactive TUI.
package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"

	"tustat/internal/render"
	"tustat/internal/status"
	"tustat/internal/tui"
)

// Version is the program version.
const Version = "0.4.0"

// Env carries the process environment so Run is testable.
type Env struct {
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Getenv    func(string) string
	StdoutTTY bool
	StdinTTY  bool
	Width     int // terminal width override; 0 detects
	Hostname  string
}

// Run executes tustat and returns the process exit code.
func Run(args []string, env Env) int {
	if env.Getenv == nil {
		env.Getenv = os.Getenv
	}
	o, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(env.Stderr, "tustat: %s\n\n%s", err, usageText)
		return 2
	}
	if o.help {
		fmt.Fprint(env.Stdout, usageText)
		return 0
	}
	if o.version {
		fmt.Fprintf(env.Stdout, "tustat %s\n", Version)
		return 0
	}

	base := status.DefaultOptions()
	base.Getenv = env.Getenv
	base.Home = env.Getenv("HOME")
	base.Check = o.check
	base.All = o.all
	base.Days = o.days

	if !o.print && !o.any && env.StdoutTTY && env.StdinTTY {
		if err := tui.Run(tui.Config{Options: base, Version: Version}); err != nil {
			fmt.Fprintf(env.Stderr, "tustat: %v\n", err)
			return 1
		}
		return 0
	}

	secs := o.sections()
	data := status.GatherSelected(base, secs)
	printStatus(env, base, data, secs)
	return 0
}

type opts struct {
	check, all, print, help, version bool
	days                             int
	want                             map[status.Section]bool
	any                              bool
}

func (o opts) sections() []status.Section {
	if !o.any {
		return status.Sections()
	}
	var out []status.Section
	for _, s := range status.Sections() {
		if o.want[s] {
			out = append(out, s)
		}
	}
	return out
}

func parseArgs(args []string) (opts, error) {
	o := opts{days: 30, want: make(map[status.Section]bool)}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--check":
			o.check = true
		case a == "--all":
			o.all = true
		case a == "--print":
			o.print = true
		case a == "--help":
			o.help = true
		case a == "--version":
			o.version = true
		case a == "--days" || a == "-d":
			i++
			if i >= len(args) {
				return o, fmt.Errorf("%s needs a number", a)
			}
			n, err := strconv.ParseUint(args[i], 10, 32)
			if err != nil {
				return o, fmt.Errorf("bad number %q", args[i])
			}
			o.days = int(n)
		case len(a) > 1 && a[0] == '-' && a[1] != '-':
			for _, ch := range a[1:] {
				switch ch {
				case 'c':
					o.check = true
				case 'a':
					o.all = true
				case 'p':
					o.print = true
				case 'h':
					o.help = true
				case 'V':
					o.version = true
				default:
					return o, fmt.Errorf("unknown option -%c", ch)
				}
			}
		default:
			s, ok := status.ParseSection(a)
			if !ok {
				return o, fmt.Errorf("unknown argument %q", a)
			}
			o.want[s] = true
			o.any = true
		}
	}
	return o, nil
}

// widthFor returns the card width: terminal width minus a small margin,
// clamped to a readable range.
func widthFor(env Env) int {
	cols := env.Width
	if cols == 0 && env.StdoutTTY {
		if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil {
			cols = w
		}
	}
	if cols == 0 {
		if n, err := strconv.Atoi(env.Getenv("COLUMNS")); err == nil {
			cols = n
		}
	}
	if cols == 0 {
		cols = 80
	}
	return clamp(cols-2, 60, 110)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func printStatus(env Env, o status.Options, data status.Data, secs []status.Section) {
	r := lipgloss.NewRenderer(env.Stdout)
	v := render.NewView(render.NewTheme(r), o.Now, o.Home, o.All)
	width := widthFor(env)

	host := env.Hostname
	if host == "" {
		host, _ = os.Hostname()
	}
	if host == "" {
		host = "localhost"
	}

	var lines []string
	lines = append(lines, v.HeaderLine(host, "v"+Version))
	lines = append(lines, "")
	for _, s := range secs {
		lines = append(lines, v.SummaryRow(s, status.Summarize(s, data), 0, false, width))
	}
	for _, s := range secs {
		lines = append(lines, "")
		lines = append(lines, v.Section(s, data, width)...)
	}
	lines = append(lines, "")

	out := strings.Join(lines, "\n")
	fmt.Fprint(env.Stdout, out)
}

const usageText = `Usage: tustat [options] [image] [flatpak] [appimage] [timers]

Shows when updates were last applied on tbzos, and what was applied.
On a terminal with no sections given, opens an interactive menu.
Otherwise prints the requested sections (all if none) and exits.

  -p, --print      print instead of opening the menu
  -c, --check      also check for available updates (network; no root)
  -a, --all        show every package / day / log line, not just the latest
  -d, --days N     Flatpak history window in days (default 30)
  -V, --version    print version
  -h, --help       this help

Menu keys:
  ↑↓ / j k  move or scroll     ⏎ / 1-5  open a section
  esc       back to the menu   c        check for available updates
  r         refresh            ?        this help      q  quit

Env:
  TUSTAT_APPIMAGE_DIRS   colon-separated AppImage dirs
                         (default ~/AppImages:~/Applications:~/.local/bin)
  NO_COLOR               disable colour
`
