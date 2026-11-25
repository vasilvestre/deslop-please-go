// Package lint finds the architectural problems of loaded packages: deep
// modules and narrow interfaces, as A Philosophy of Software Design has them.
package lint

import (
	"sort"

	"github.com/vasilvestre/deslop-please-go/internal/model"
	"github.com/vasilvestre/deslop-please-go/internal/report"
)

// Run is every lint over m, ordered by lint, then key.
func Run(m *model.Model) []report.Diagnostic {
	x := buildIndex(m)
	var diagnostics []report.Diagnostic
	for _, lint := range []func(*index) []report.Diagnostic{moduleCycles, tangles, fieldLeaks, shallowModules} {
		diagnostics = append(diagnostics, lint(x)...)
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		a, b := diagnostics[i], diagnostics[j]
		if a.Lint != b.Lint {
			return order(a.Lint) < order(b.Lint)
		}
		return a.Key < b.Key
	})
	return diagnostics
}

func order(lint report.Lint) int {
	for i, l := range []report.Lint{report.ModuleCycle, report.Tangle, report.FieldLeak, report.ShallowModule} {
		if l == lint {
			return i
		}
	}
	return -1
}
