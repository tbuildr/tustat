package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGatherImage(t *testing.T) {
	bin := t.TempDir()
	home := t.TempDir()
	writeScript(t, bin, "rpm-ostree", `#!/bin/sh
case "$1" in
  status) cat <<'JSON'
{"deployments":[
 {"booted":true,"version":"41.20251101.0","timestamp":1793523600,
  "container-image-reference":"ostree-image-signed:docker://ghcr.io/ublue-os/bazzite-gnome:stable",
  "container-image-reference-digest":"sha256:abcdef0123456789","checksum":"aaa"},
 {"staged":true,"version":"41.20251105.0","timestamp":1793962000,"checksum":"bbb"},
 {"booted":false,"version":"41.20251020.0","timestamp":1791000000,"checksum":"prev"}
]}
JSON
  ;;
  db) printf '%s\n' "Upgraded:" "  bash 5.2.37-1.fc41 -> 5.2.38-1.fc41" ;;
  upgrade) echo "No upgrade available." ;;
esac
`)
	writeScript(t, bin, "journalctl", `#!/bin/sh
echo "1794002400.5 Bootloader updated; bootconfig swap: yes"
`)

	opts := fakeEnv(t, bin, home)
	opts.Check = true
	im := GatherImage(opts)

	if im.Err != "" {
		t.Fatalf("unexpected error: %s", im.Err)
	}
	if im.Booted.Version != "41.20251101.0" || im.Booted.Checksum != "aaa" {
		t.Errorf("booted = %+v", im.Booted)
	}
	if im.Booted.Ref != "ostree-image-signed:docker://ghcr.io/ublue-os/bazzite-gnome:stable" {
		t.Errorf("ref = %q", im.Booted.Ref)
	}
	if im.Booted.Digest != "sha256:abcdef0123456789" {
		t.Errorf("digest = %q", im.Booted.Digest)
	}
	if im.Booted.Built.Unix() != 1793523600 {
		t.Errorf("built = %v", im.Booted.Built)
	}
	if im.Applied == nil || im.Applied.Unix() != 1794002400 {
		t.Errorf("applied = %v; want journal timestamp", im.Applied)
	}
	if im.Prev == nil || im.Prev.Version != "41.20251020.0" {
		t.Errorf("prev = %+v", im.Prev)
	}
	if len(im.Changes) != 1 || im.Changes[0].Name != "bash" || im.Changes[0].From != "5.2.37-1.fc41" {
		t.Errorf("changes = %#v", im.Changes)
	}
	if im.Staged == nil || im.Staged.Version != "41.20251105.0" {
		t.Errorf("staged = %+v", im.Staged)
	}
	if im.StagedAt == nil {
		t.Error("staged-at timestamp missing")
	}
	if im.Check == nil || *im.Check != "No upgrade available." {
		t.Errorf("check = %v", im.Check)
	}
}

func TestGatherImageWithoutRpmOstree(t *testing.T) {
	bin := t.TempDir()
	opts := fakeEnv(t, bin, t.TempDir())
	im := GatherImage(opts)
	if im.Err != "rpm-ostree not found" {
		t.Fatalf("err = %q", im.Err)
	}
}

func TestGatherTimers(t *testing.T) {
	bin := t.TempDir()
	home := t.TempDir()
	writeScript(t, bin, "systemctl", `#!/bin/sh
result=0
prev=""
for a in "$@"; do
  if [ "$prev" = "-P" ]; then result=1; fi
  unit="$a"
  prev="$a"
done
if [ "$result" = 1 ]; then
  case "$unit" in
    uupd.service) echo success ;;
    *) echo failed ;;
  esac
  exit 0
fi
case "$unit" in
  uupd.timer) printf '%s\n' "LoadState=loaded" "UnitFileState=enabled" \
    "LastTriggerUSec=@1793980800" "NextElapseUSecRealtime=@1794106800" "Unit=uupd.service" ;;
  flatpak-user-update.timer) printf '%s\n' "LoadState=loaded" "UnitFileState=disabled" \
    "LastTriggerUSec=@1793790000" "NextElapseUSecRealtime=@1794106800" "Unit=flatpak-user.service" ;;
  *) echo "LoadState=not-found" ;;
esac
`)
	timers := GatherTimers(fakeEnv(t, bin, home))
	if len(timers) != 2 {
		t.Fatalf("got %d timers: %#v", len(timers), timers)
	}
	if timers[0].Name != "uupd.timer" || timers[0].Last == nil || timers[0].Next == nil {
		t.Errorf("uupd = %+v", timers[0])
	}
	if timers[1].Enabled != "disabled" || !timers[1].User {
		t.Errorf("flatpak user timer = %+v", timers[1])
	}
	if got := TimersFailing(timers); got != 1 {
		t.Errorf("failing = %d; want 1", got)
	}
	if latest := LatestRun(timers); latest == nil || latest.Unix() != 1793980800 {
		t.Errorf("latest = %v", latest)
	}
}

func TestGatherFlatpak(t *testing.T) {
	bin := t.TempDir()
	home := t.TempDir()
	writeScript(t, bin, "flatpak", `#!/bin/sh
case "$1" in
  history) printf '%s\n' \
    "Oct 20 11:05:00	update	org.example.App	stable	system" \
    "Oct 12 09:15:02	update	org.example.App	stable	system" \
    "Oct 12 09:15:03	update	org.other.Tool	stable	user" \
    "Oct 14 10:20:00	install	org.new.App	stable	system" ;;
  list) printf '%s\n' "org.example.App	1.2.3" "org.other.Tool	0.9.1" ;;
  remote-ls) printf '%s\n' "org.example.App	1.3.0	stable" ;;
esac
`)
	opts := fakeEnv(t, bin, home)
	opts.Check = true
	fp := GatherFlatpak(opts)
	if fp.Err != "" {
		t.Fatalf("err = %s", fp.Err)
	}
	if len(fp.Entries) != 4 {
		t.Fatalf("got %d entries", len(fp.Entries))
	}
	if got := fp.Entries[len(fp.Entries)-1].Day; got != "2026-10-20" {
		t.Errorf("last day = %q", got)
	}
	if fp.Versions["org.example.App"] != "1.2.3" {
		t.Errorf("versions = %v", fp.Versions)
	}
	if fp.Check == nil || !strings.Contains(*fp.Check, "1.3.0") {
		t.Errorf("check = %v", fp.Check)
	}
}

func TestGatherAppImagesSnapshot(t *testing.T) {
	bin := t.TempDir()
	home := t.TempDir()
	appDir := filepath.Join(home, "AppImages")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	apps := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(appDir, "alpha.AppImage")
	if err := os.WriteFile(a, []byte("alpha-v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 11, 1, 10, 0, 0, 0, time.Local)
	if err := os.Chtimes(a, base, base); err != nil {
		t.Fatal(err)
	}
	desktop := "[Desktop Entry]\nName=Alpha\nX-AppImage-Version=1.0\nExec=" + a + "\n"
	if err := os.WriteFile(filepath.Join(apps, "alpha.desktop"), []byte(desktop), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := fakeEnv(t, bin, home)
	opts.Days = 30

	first := GatherAppImages(opts)
	if !first.FirstRun {
		t.Fatal("first gather should record a baseline")
	}
	if len(first.Items) != 1 {
		t.Fatalf("items = %#v", first.Items)
	}
	if first.Items[0].Name != "Alpha" || first.Items[0].Version != "1.0" {
		t.Errorf("desktop metadata = %+v", first.Items[0])
	}
	if len(first.LogEvents) != 0 {
		t.Errorf("first run wrote log events: %#v", first.LogEvents)
	}
	snap := filepath.Join(home, "state", "tustat", "appimages.tsv")
	if _, err := os.Stat(snap); err != nil {
		t.Fatalf("snapshot not written: %v", err)
	}

	// Touch the file and change its content: the second pass must log an
	// update, not an add.
	later := base.Add(48 * time.Hour)
	if err := os.WriteFile(a, []byte("alpha-v2-longer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(a, later, later); err != nil {
		t.Fatal(err)
	}
	second := GatherAppImages(opts)
	if second.FirstRun {
		t.Fatal("second gather must not be a first run")
	}
	if len(second.LogEvents) != 1 || second.LogEvents[0].Kind != "updated" || second.LogEvents[0].File != "alpha.AppImage" {
		t.Fatalf("log events = %#v", second.LogEvents)
	}
	if !strings.Contains(second.LogEvents[0].Detail, "1.0 -> 1.0") {
		t.Errorf("detail = %q", second.LogEvents[0].Detail)
	}

	// Removing the file logs a removal.
	if err := os.Remove(a); err != nil {
		t.Fatal(err)
	}
	third := GatherAppImages(opts)
	if len(third.Items) != 0 {
		t.Fatalf("items after removal = %#v", third.Items)
	}
	found := false
	for _, ev := range third.LogEvents {
		if ev.Kind == "removed" && ev.File == "alpha.AppImage" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no removal event in %#v", third.LogEvents)
	}
	// The removal line was appended after the update, so both are visible.
	if len(third.LogEvents) < 2 {
		t.Errorf("expected update and removal events, got %#v", third.LogEvents)
	}
}
