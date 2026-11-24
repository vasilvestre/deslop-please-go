package allow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vsilvestre/deslop-please-go/internal/allow"
	"github.com/vsilvestre/deslop-please-go/internal/report"
)

func diag(lint report.Lint, key string) report.Diagnostic {
	return report.Diagnostic{Lint: lint, Key: key}
}

func TestRoundTripIsSortedAndStable(t *testing.T) {
	diagnostics := []report.Diagnostic{
		diag(report.Tangle, "b.B,b.C"),
		diag(report.FieldLeak, "q.Q.Items"),
		diag(report.FieldLeak, "q.Q.Items"),
		diag(report.ModuleCycle, "a,b"),
	}
	path := filepath.Join(t.TempDir(), "allow")
	if err := allow.Of(diagnostics).Write(path); err != nil {
		t.Fatal(err)
	}
	text, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(text)), "\n")
	want := []string{"field-leak q.Q.Items", "module-cycle a,b", "tangle b.B,b.C"}
	if got := lines[2:]; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q", got)
	}
	list, err := allow.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diagnostics {
		if !list.Allows(d) {
			t.Errorf("%s %s not allowed after the round trip", d.Lint, d.Key)
		}
	}
	if list.Allows(diag(report.FieldLeak, "q.Q.Head")) || list.Fixed(diagnostics) != 0 {
		t.Error("an unlisted diagnostic is allowed, or a present entry counts as fixed")
	}
}

func TestShrunkCircleStaysAllowedAndFixedEntriesAreCounted(t *testing.T) {
	list, err := allow.Parse("# note\ntangle a.A,a.B,a.C\nshallow-module store\n")
	if err != nil {
		t.Fatal(err)
	}
	if !list.Allows(diag(report.Tangle, "a.A,a.B")) {
		t.Error("a smaller circle is an improvement")
	}
	if list.Allows(diag(report.Tangle, "a.A,a.D")) {
		t.Error("a circle with a new member is new debt")
	}
	if got := list.Fixed([]report.Diagnostic{diag(report.Tangle, "a.A,a.B")}); got != 1 {
		t.Errorf("fixed = %d, want 1", got)
	}
}

func TestInvalidEntriesNameTheirLine(t *testing.T) {
	_, err := allow.Parse("tangle a\nnonsense b\n")
	if err == nil || !strings.Contains(err.Error(), "2:") {
		t.Fatalf("err = %v", err)
	}
}
