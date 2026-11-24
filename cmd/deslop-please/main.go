// Command deslop-please is an opinionated linter that removes sloppy Go code
// structure: deep modules, narrow interfaces.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vsilvestre/deslop-please-go/internal/allow"
	"github.com/vsilvestre/deslop-please-go/internal/lint"
	"github.com/vsilvestre/deslop-please-go/internal/model"
	"github.com/vsilvestre/deslop-please-go/internal/render"
	"github.com/vsilvestre/deslop-please-go/internal/report"
)

const usage = `usage: deslop-please [--allow <file>] [--update-allow <file>] [patterns...]

Runs architectural lints (module-cycle, tangle, field-leak, shallow-module)
on the Go packages that match patterns, ./... by default.

options:
  --allow <file>         mute the listed existing debt; new errors still fail
  --update-allow <file>  write the current diagnostics to <file>
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("deslop-please", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprint(stderr, usage) }
	allowPath := flags.String("allow", "", "")
	updatePath := flags.String("update-allow", "", "")
	var patterns []string
	for rest := args; ; {
		if err := flags.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return 0
			}
			return 2
		}
		if flags.NArg() == 0 {
			break
		}
		patterns = append(patterns, flags.Arg(0))
		rest = flags.Args()[1:]
	}
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	m, err := model.Load(".", patterns)
	if err != nil {
		fmt.Fprintf(stderr, "deslop-please: %v\n", err)
		return 2
	}
	diagnostics := lint.Run(m)
	if *updatePath != "" {
		return update(*updatePath, diagnostics, stdout, stderr)
	}
	return check(*allowPath, diagnostics, stdout, stderr)
}

func update(path string, diagnostics []report.Diagnostic, stdout, stderr io.Writer) int {
	list := allow.Of(diagnostics)
	if err := list.Write(path); err != nil {
		fmt.Fprintf(stderr, "deslop-please: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "deslop-please: wrote %d entries to %s\n", list.Len(), path)
	return 0
}

func check(allowPath string, diagnostics []report.Diagnostic, stdout, stderr io.Writer) int {
	fixed := 0
	if allowPath != "" {
		list, err := allow.Read(allowPath)
		if err != nil {
			fmt.Fprintf(stderr, "deslop-please: %v\n", err)
			return 2
		}
		fixed = list.Fixed(diagnostics)
		var remaining []report.Diagnostic
		for _, d := range diagnostics {
			if !list.Allows(d) {
				remaining = append(remaining, d)
			}
		}
		diagnostics = remaining
	}
	fmt.Fprint(stdout, render.Text(diagnostics, fixed, allowPath, colored()))
	if len(diagnostics) > 0 {
		return 1
	}
	return 0
}

func colored() bool {
	switch os.Getenv("CLICOLOR_FORCE") {
	case "", "0":
	default:
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
