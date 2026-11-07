# tustat

Update status for tbzos (Bazzite-based): when image, Flatpak and AppImage updates
were last applied, and what was applied. Companion to `tu`.

Read-only, no root. The only thing it writes is the AppImage snapshot and change log
in `${XDG_STATE_HOME:-~/.local/state}/tustat/` (same format as the old bash version).

## Build

Needs Go 1.24 or newer.

```bash
make build            # CGO_ENABLED=0, fully static binary
# or
go build -o tustat ./cmd/tustat
install -Dm755 tustat ~/.local/bin/tustat
```

`CGO_ENABLED=0 go build -o tustat ./cmd/tustat` produces a static binary that runs
on any Linux host, independent of the C library.

## Use

```bash
tustat                 # interactive menu on a terminal
tustat -p              # print instead of opening the menu
tustat image flatpak   # just these
tustat -c              # also check what's available (network)
tustat -a              # every package / day / log line
tustat -d 90 flatpak   # wider Flatpak window
```

Without a terminal (piped output), tustat always uses print mode.

### TUI

The interactive view is built with Bubble Tea, bubbles, lipgloss and harmonica:

- a menu with one summary row per section (freshness dot, age, headline),
  grouped by nothing but order, with a spring-animated selection bar;
- each section opens as a scrollable card: booted image and applied changes,
  Flatpak history and update checks, AppImage inventory and change log,
  automatic-update timers;
- `1`–`5` jump straight to a section, `5` being "Everything" on one page;
- a spinner marks long gathers; checks never fabricate progress.

Keys:

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | Move / scroll |
| `PgUp`/`PgDn` | Page the detail view |
| `g`, `G` | Top / bottom |
| `Enter` | Open a section |
| `1`–`5` | Jump to a section |
| `Esc` / `h` | Back to the menu |
| `r` | Refresh what is shown |
| `c` | Check for available updates (network) |
| `?` | Help |
| `q` | Quit |

Colours are semantic: green under a week old, yellow under a month, red older;
`NO_COLOR` disables colour everywhere.

## Where the data comes from

| Section   | Source |
|-----------|--------|
| Image     | `rpm-ostree status --json`; applied time from `ostree-finalize-staged` in the journal; changes from `rpm-ostree db diff` (rollback → booted, booted → staged) |
| Flatpak   | `flatpak history` (journal) + `flatpak list` for current versions |
| AppImage  | scans `~/AppImages`, `~/Applications`, `~/.local/bin` (override with `TUSTAT_APPIMAGE_DIRS`); name/version from Gear Lever `.desktop` files; own snapshot diff for history |
| Timers    | `systemctl show` for uupd / ublue-update / rpm-ostreed-automatic / flatpak update timers |

The AppImage snapshot (`appimages.tsv`) and change log (`appimage-history.log`)
keep the format written by the bash and Zig versions, so history carries over.

## Development

```bash
make test     # go test ./...
make vet      # go vet ./...
```
