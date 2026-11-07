// Command tustat shows when updates were last applied on tbzos: the booted
// image, Flatpak, AppImages and the automatic-update timers.
package main

import (
	"os"

	"github.com/charmbracelet/x/term"

	"tustat/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Getenv:    os.Getenv,
		StdoutTTY: term.IsTerminal(os.Stdout.Fd()),
		StdinTTY:  term.IsTerminal(os.Stdin.Fd()),
	}))
}
