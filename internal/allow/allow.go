// Package allow is the plain-text list of existing architectural debt: one
// `<lint> <key>` per line, sorted.
package allow

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vasilvestre/deslop-please-go/internal/report"
)

const header = "# Existing architectural debt, muted by `deslop-please --allow`.\n" +
	"# Fixing an entry never fails the check; new issues not listed here do.\n"

type entry struct {
	lint report.Lint
	key  string
}

type List struct{ entries []entry }

// Of lists what diagnostics report.
func Of(diagnostics []report.Diagnostic) *List {
	seen := map[entry]bool{}
	list := &List{}
	for _, d := range diagnostics {
		if e := (entry{d.Lint, d.Key}); !seen[e] {
			seen[e] = true
			list.entries = append(list.entries, e)
		}
	}
	sort.Slice(list.entries, func(i, j int) bool {
		a, b := list.entries[i], list.entries[j]
		return a.lint < b.lint || a.lint == b.lint && a.key < b.key
	})
	return list
}

func Read(path string) (*List, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	list, err := Parse(string(text))
	if err != nil {
		return nil, fmt.Errorf("%s:%w", path, err)
	}
	return list, nil
}

func Parse(text string) (*List, error) {
	list := &List{}
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, key, _ := strings.Cut(line, " ")
		lint, ok := report.ParseLint(name)
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("%d: invalid allowlist entry", i+1)
		}
		list.entries = append(list.entries, entry{lint, strings.TrimSpace(key)})
	}
	return list, nil
}

func (l *List) Write(path string) error {
	var out strings.Builder
	out.WriteString(header)
	for _, e := range l.entries {
		fmt.Fprintf(&out, "%s %s\n", e.lint, e.key)
	}
	return os.WriteFile(path, []byte(out.String()), 0o644)
}

func (l *List) Len() int { return len(l.entries) }

// Allows is whether an entry mutes d. Shrinking an allowed circle is an
// improvement, never a new error, so an entry mutes every tangle or module
// cycle whose members are among its own.
func (l *List) Allows(d report.Diagnostic) bool {
	for _, e := range l.entries {
		if e.covers(d) {
			return true
		}
	}
	return false
}

// Fixed counts the entries that mute nothing of diagnostics any more.
func (l *List) Fixed(diagnostics []report.Diagnostic) int {
	fixed := 0
	for _, e := range l.entries {
		covered := false
		for _, d := range diagnostics {
			covered = covered || e.covers(d)
		}
		if !covered {
			fixed++
		}
	}
	return fixed
}

func (e entry) covers(d report.Diagnostic) bool {
	if d.Lint != e.lint {
		return false
	}
	if d.Key == e.key {
		return true
	}
	if e.lint != report.Tangle && e.lint != report.ModuleCycle {
		return false
	}
	allowed := map[string]bool{}
	for _, member := range strings.Split(e.key, ",") {
		allowed[member] = true
	}
	for _, member := range strings.Split(d.Key, ",") {
		if !allowed[member] {
			return false
		}
	}
	return true
}
