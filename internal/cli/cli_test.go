package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tustat/internal/status"
)

func TestParseArgs(t *testing.T) {
	o, err := parseArgs([]string{"-c", "image", "-a", "-d", "90", "flatpak"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.check || !o.all || o.days != 90 {
		t.Errorf("flags = %+v", o)
	}
	if got := o.sections(); len(got) != 2 || got[0] != status.SectionImage || got[1] != status.SectionFlatpak {
		t.Errorf("sections = %v", got)
	}

	o, err = parseArgs([]string{"--print", "--days=7"})
	if err == nil {
		t.Fatal("--days=7 should be rejected like the original parser")
	}

	o, err = parseArgs([]string{"--print", "timers"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.print || len(o.sections()) != 1 {
		t.Errorf("o = %+v", o)
	}
}

func TestParseArgsErrors(t *testing.T) {
	for _, args := range [][]string{
		{"nope"},
		{"-x"},
		{"-d"},
		{"-d", "many"},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%v) accepted invalid input", args)
		}
	}
}

func TestRunHelpAndVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	env := Env{Stdout: &out, Stderr: &errOut}
	if code := Run([]string{"--help"}, env); code != 0 {
		t.Fatalf("help exit = %d", code)
	}
	if !strings.Contains(out.String(), "Usage: tustat") {
		t.Errorf("help output = %q", out.String())
	}

	out.Reset()
	if code := Run([]string{"-V"}, env); code != 0 {
		t.Fatalf("version exit = %d", code)
	}
	if strings.TrimSpace(out.String()) != "tustat "+Version {
		t.Errorf("version output = %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := Run([]string{"bogus"}, env); code != 2 {
		t.Fatalf("bad arg exit = %d; want 2", code)
	}
	if !strings.Contains(errOut.String(), "unknown argument") || !strings.Contains(errOut.String(), "Usage:") {
		t.Errorf("stderr = %q", errOut.String())
	}
}

func TestRunPrintWithFixtures(t *testing.T) {
	bin := t.TempDir()
	home := t.TempDir()
	scripts := map[string]string{
		"rpm-ostree": `#!/bin/sh
case "$1" in
  status) printf '%s' '{"deployments":[
   {"booted":true,"version":"41.1","timestamp":1793523600,"checksum":"a"},
   {"booted":false,"version":"41.0","timestamp":1791000000,"checksum":"b"}]}' ;;
  db) printf '%s\n' "Upgraded:" "  bash 5.2.37-1.fc41 -> 5.2.38-1.fc41" ;;
esac
`,
		"journalctl": `#!/bin/sh
echo "1794002400.5 Bootloader updated"
`,
		"flatpak": `#!/bin/sh
case "$1" in
  history) printf '%s\n' "Oct 20 11:05:00	update	org.example.App	stable	system" ;;
  list) printf '%s\n' "org.example.App	1.2.3" ;;
esac
`,
		"systemctl": `#!/bin/sh
echo "LoadState=not-found"
`,
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	var out, errOut bytes.Buffer
	env := Env{
		Stdout: &out,
		Stderr: &errOut,
		Width:  90,
		Getenv: func(k string) string {
			switch k {
			case "PATH":
				return bin + ":" + os.Getenv("PATH")
			case "HOME":
				return home
			case "XDG_STATE_HOME":
				return filepath.Join(home, "state")
			}
			return ""
		},
	}
	code := Run([]string{"-p"}, env)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	text := out.String()
	for _, want := range []string{"IMAGE", "FLATPAK", "APPIMAGE", "AUTO-UPDATES"} {
		if !strings.Contains(text, want) {
			t.Errorf("print output missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "\x1b[") {
		t.Error("print output contains ANSI escapes on a non-TTY")
	}
	if !strings.Contains(text, "bash") || !strings.Contains(text, "→") {
		t.Errorf("print output missing the diff:\n%s", text)
	}
}

func TestRunPrintOnlyRequestedSection(t *testing.T) {
	bin := t.TempDir()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "flatpak"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	env := Env{
		Stdout: &out,
		Width:  80,
		Getenv: func(k string) string {
			switch k {
			case "PATH":
				return bin + ":" + os.Getenv("PATH")
			case "HOME":
				return home
			case "XDG_STATE_HOME":
				return filepath.Join(home, "state")
			}
			return ""
		},
	}
	if code := Run([]string{"-p", "flatpak"}, env); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	text := out.String()
	if strings.Contains(text, "AUTO-UPDATES") || strings.Contains(text, "◆ IMAGE") {
		t.Errorf("only Flatpak was requested:\n%s", text)
	}
	if !strings.Contains(text, "▣ FLATPAK") {
		t.Errorf("Flatpak card missing:\n%s", text)
	}
}
