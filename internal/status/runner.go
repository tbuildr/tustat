package status

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// collector runs external commands for one gather.
type collector struct {
	opts Options
}

// run executes argv without a shell and returns its stdout, stderr and
// success. Output is capped to keep a runaway command from eating memory.
func (c *collector) run(name string, args ...string) (stdout, stderr string, ok bool) {
	// exec.Command resolves the binary against the process PATH, so resolve
	// against the collector's PATH first: that is what lookups use and what
	// tests inject.
	if !strings.ContainsRune(name, '/') {
		resolved, found := lookIn(c.opts.Getenv("PATH"), name)
		if !found {
			return "", "", false
		}
		name = resolved
	}
	cmd := exec.Command(name, args...)
	// The options' PATH is authoritative: it is what lookups used and what
	// tests inject.
	if path := c.opts.Getenv("PATH"); path != "" {
		env := make([]string, 0, len(os.Environ())+1)
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "PATH=") {
				env = append(env, e)
			}
		}
		cmd.Env = append(env, "PATH="+path)
	}
	var ob, eb bytes.Buffer
	ob.Grow(4096)
	cmd.Stdout = &limitedWriter{w: &ob, n: 16 << 20}
	cmd.Stderr = &limitedWriter{w: &eb, n: 1 << 20}
	err := cmd.Run()
	return ob.String(), eb.String(), err == nil
}

// limitedWriter stops accepting writes after n bytes.
type limitedWriter struct {
	w *bytes.Buffer
	n int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if left := l.n - l.w.Len(); left <= 0 {
		return len(p), nil
	} else if len(p) > left {
		l.w.Write(p[:left])
		return len(p), nil
	}
	l.w.Write(p)
	return len(p), nil
}

// have reports whether name resolves on the configured PATH.
func (c *collector) have(name string) bool {
	if strings.ContainsRune(name, '/') {
		return false
	}
	_, found := lookIn(c.opts.Getenv("PATH"), name)
	return found
}

// lookIn finds an executable called name in a colon-separated path.
func lookIn(path, name string) (string, bool) {
	for _, dir := range strings.Split(path, ":") {
		if dir == "" {
			continue
		}
		full := dir + "/" + name
		if info, err := os.Stat(full); err == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			return full, true
		}
	}
	return "", false
}

// journalLast returns the timestamp of the newest journal line for unit
// matching pattern, or nil when there is none.
func (c *collector) journalLast(unit, pattern string) *time.Time {
	stdout, _, _ := c.run("journalctl", "-u", unit, "-o", "short-unix", "-q", "-r", "-n", "1", "--no-pager", "-g", pattern)
	line := strings.Trim(stdout, " \n\t")
	if line == "" {
		return nil
	}
	end := strings.IndexAny(line, ". ")
	if end < 0 {
		end = len(line)
	}
	secs, err := strconv.ParseInt(line[:end], 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(secs, 0)
	return &t
}

// prop returns the value of key in systemctl's Key=Value output.
func prop(text, key string) string {
	for _, line := range strings.Split(text, "\n") {
		if len(line) > len(key) && strings.HasPrefix(line, key) && line[len(key)] == '=' {
			return line[len(key)+1:]
		}
	}
	return ""
}

// unixProp parses systemctl's --timestamp=unix form, "@1712345678".
func unixProp(v string) *time.Time {
	if len(v) < 2 || v[0] != '@' {
		return nil
	}
	secs, err := strconv.ParseInt(v[1:], 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(secs, 0)
	return &t
}

// trimOutput trims spaces and newlines, as the command output wants.
func trimOutput(s string) string { return strings.Trim(s, " \n") }
