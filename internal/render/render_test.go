package render_test

import (
	"strings"
	"testing"

	"github.com/vasilvestre/deslop-please-go/internal/render"
	"github.com/vasilvestre/deslop-please-go/internal/report"
)

func TestPicturesAndSummary(t *testing.T) {
	at := func(file string, line int) report.Location { return report.Location{File: file, Line: line, Col: 1} }
	diagnostics := []report.Diagnostic{
		{Lint: report.ModuleCycle, Message: "2 modules in `.` use each other in a circle", At: &report.Location{File: "a/x.go", Line: 3, Col: 2},
			Notes: []string{"members: a, b"},
			Picture: report.Circle{Steps: []report.Step{
				{Member: "a", At: at("a/x.go", 3), Uses: "X uses Y"},
				{Member: "b", At: at("b/net/z.go", 5), Uses: "Z uses W"},
			}, Rest: "not drawn: 1 more tie"}},
		{Lint: report.FieldLeak, Message: "`Items` is known outside `Q`",
			Picture: report.Spread{Owner: "q.Q", Field: "Items",
				Packages: []report.Count{{N: 12, Name: "sched"}, {N: 3, Name: "app"}}, Others: []string{"kept: limit"}}},
		{Lint: report.ShallowModule, Message: "`store` is shallow",
			Picture: report.Lists{Shown: 50, Hidden: 3, Long: []report.Count{{N: 9, Name: "Store"}}}},
	}
	want := []string{
		"error[module-cycle]: 2 modules in `.` use each other in a circle",
		"  --> a/x.go:3:2",
		"  = note: members: a, b",
		"    ┌─▶ a",
		"    │     ┃ a/x.go:3      X uses Y",
		"    │     ▼",
		"    │   b",
		"    │     │ b/net/z.go:5  Z uses W",
		"    └─────┘",
		"    not drawn: 1 more tie",
		"",
		"error[field-leak]: `Items` is known outside `Q`",
		"    q.Q",
		"      Items ◀━┳━ 12  sched",
		"              ┗━  3  app",
		"      kept: limit",
		"",
		"error[shallow-module]: `store` is shallow",
		"                  7",
		"    shows  ███████│█████████████████████████████████⋯ 50",
		"    hides  ███    │ 3",
		"    Store  ███████│██ 9",
		"",
		"deslop-please: 3 errors, well done!",
		"",
	}
	if got := render.Text(diagnostics, 0, "", false); got != strings.Join(want, "\n") {
		t.Fatalf("got\n%s\nwant\n%s", got, strings.Join(want, "\n"))
	}
}

func TestNothingToReport(t *testing.T) {
	if got := render.Text(nil, 2, ".allow", false); !strings.HasSuffix(got, "deslop-please: ok\n") ||
		!strings.Contains(got, "2 allowed entries in .allow are now fixed") {
		t.Fatalf("got %q", got)
	}
}
