// Package render writes diagnostics the way rustc and clippy do, in color
// when asked, with a picture under each report.
package render

import (
	"fmt"
	"strings"

	"github.com/vsilvestre/deslop-please-go/internal/report"
)

const tool = "deslop-please"

type paint bool

func (p paint) sgr(code, text string) string {
	if !p {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (p paint) red(text string) string   { return p.sgr("1;31", text) }
func (p paint) blue(text string) string  { return p.sgr("1;34", text) }
func (p paint) green(text string) string { return p.sgr("1;32", text) }
func (p paint) bold(text string) string  { return p.sgr("1", text) }

// Text renders diagnostics, then a summary. fixed is how many entries of the
// allowlist at allowPath mute nothing any more.
func Text(diagnostics []report.Diagnostic, fixed int, allowPath string, color bool) string {
	p := paint(color)
	var out strings.Builder
	for _, d := range diagnostics {
		fmt.Fprintf(&out, "%s%s\n", p.red("error["+string(d.Lint)+"]"), p.bold(": "+d.Message))
		if d.At != nil {
			fmt.Fprintf(&out, "  %s %s\n", p.blue("-->"), d.At)
		}
		for _, note := range d.Notes {
			fmt.Fprintf(&out, "  %s %s %s\n", p.blue("="), p.bold("note:"), note)
		}
		for _, line := range draw(d.Picture, p) {
			fmt.Fprintf(&out, "    %s\n", line)
		}
		out.WriteString("\n")
	}
	if fixed > 0 {
		fmt.Fprintf(&out, "%s %d allowed %s in %s are now fixed (run `%s --update-allow %s` to tighten it)\n",
			p.bold("note:"), fixed, plural(fixed), allowPath, tool, allowPath)
	}
	if len(diagnostics) == 0 {
		fmt.Fprintf(&out, "%s: %s\n", tool, p.green("ok"))
		return out.String()
	}
	noun := "errors"
	if len(diagnostics) == 1 {
		noun = "error"
	}
	fmt.Fprintf(&out, "%s: %s, well done!\n", tool, p.red(fmt.Sprintf("%d %s", len(diagnostics), noun)))
	return out.String()
}

func plural(n int) string {
	if n == 1 {
		return "entry"
	}
	return "entries"
}
