package lint_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vasilvestre/deslop-please-go/internal/lint"
	"github.com/vasilvestre/deslop-please-go/internal/model"
	"github.com/vasilvestre/deslop-please-go/internal/report"
)

func run(t *testing.T, fixture string) []report.Diagnostic {
	t.Helper()
	m, err := model.Load(filepath.Join("testdata", fixture), []string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	return lint.Run(m)
}

func keys(diagnostics []report.Diagnostic) []string {
	var keys []string
	for _, d := range diagnostics {
		keys = append(keys, string(d.Lint)+" "+d.Key)
	}
	return keys
}

func expect(t *testing.T, fixture string, want ...string) []report.Diagnostic {
	t.Helper()
	got := run(t, fixture)
	if strings.Join(keys(got), "\n") != strings.Join(want, "\n") {
		t.Fatalf("%s: got\n%s\nwant\n%s", fixture, strings.Join(keys(got), "\n"), strings.Join(want, "\n"))
	}
	return got
}

func TestModuleCycleBetweenSiblingSubtrees(t *testing.T) {
	d := expect(t, "cycle", "module-cycle a,b")
	if d[0].Message != "2 modules in `.` use each other in a circle" {
		t.Fatal(d[0].Message)
	}
	circle := d[0].Picture.(report.Circle)
	if len(circle.Steps) != 2 {
		t.Fatalf("steps: %v", circle.Steps)
	}
}

func TestTangleAcrossFilesOnly(t *testing.T) {
	d := expect(t, "tangle", "tangle k.Tasks,k.Timers")
	if !strings.Contains(d[0].Message, "across 2 files") {
		t.Fatal(d[0].Message)
	}
}

func TestFieldLeakOfAGuardedStructOnly(t *testing.T) {
	d := expect(t, "leak", "field-leak queue.RunQueue.Items")
	spread := d[0].Picture.(report.Spread)
	if got := strings.Join(spread.Others, "|"); got != "kept: limit" {
		t.Fatalf("others: %s", got)
	}
	if d[0].Message != "`Items` is known outside `RunQueue`: 1 place in 1 package" {
		t.Fatal(d[0].Message)
	}
}

func TestShallowModuleThatShowsMoreThanItHides(t *testing.T) {
	d := expect(t, "shallow", "shallow-module store")
	lists := d[0].Picture.(report.Lists)
	if lists.Shown != 8 || lists.Hidden != 2 {
		t.Fatalf("%+v", lists)
	}
}
