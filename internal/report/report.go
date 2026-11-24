// Package report is what the lints say and what the renderer and the
// allowlist read of it.
package report

import "fmt"

// HeldInMind is what a reader holds in mind at once (Miller, 1956). A tangle
// of more, or a list of more functions, is too much; notes list no more.
const HeldInMind = 7

type Lint string

const (
	ModuleCycle   Lint = "module-cycle"
	Tangle        Lint = "tangle"
	FieldLeak     Lint = "field-leak"
	ShallowModule Lint = "shallow-module"
)

func ParseLint(name string) (Lint, bool) {
	for _, lint := range []Lint{ModuleCycle, Tangle, FieldLeak, ShallowModule} {
		if string(lint) == name {
			return lint, true
		}
	}
	return "", false
}

type Location struct {
	File      string
	Line, Col int
}

func (l Location) String() string { return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Col) }

type Diagnostic struct {
	Lint    Lint
	Key     string // stable identifier, used by the allowlist
	Message string
	At      *Location
	Notes   []string
	Picture Picture
}

// Picture is why a report is a problem, drawn under its notes.
type Picture interface{ picture() }

// Circle: each step's member knows the next step's, the last the first's.
type Circle struct {
	Steps []Step
	Rest  string
}

type Step struct {
	Member string
	At     Location
	Uses   string
}

// Spread: a field and the packages that work on it, most places first.
type Spread struct {
	Owner, Field string
	Packages     []Count
	Others       []string
}

type Count struct {
	N    int
	Name string
}

// Lists: how many functions a package shows and hides, and each list it is
// learned by that is longer than a reader holds in mind.
type Lists struct {
	Shown, Hidden int
	Long          []Count
}

func (Circle) picture() {}
func (Spread) picture() {}
func (Lists) picture()  {}
