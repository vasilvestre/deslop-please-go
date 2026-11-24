package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func exec(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out, errs bytes.Buffer
	code := run(args, &out, &errs)
	return code, out.String() + errs.String()
}

func TestAllowlistMutesExistingDebtOnly(t *testing.T) {
	t.Chdir("../../internal/lint/testdata/cycle")
	allow := filepath.Join(t.TempDir(), "allow")

	code, out := exec(t)
	if code != 1 || !strings.Contains(out, "error[module-cycle]") || !strings.HasSuffix(out, "1 error, well done!\n") {
		t.Fatalf("code=%d\n%s", code, out)
	}
	if code, out := exec(t, "--update-allow", allow); code != 0 || !strings.Contains(out, "wrote 1 entries") {
		t.Fatalf("code=%d\n%s", code, out)
	}
	if code, out := exec(t, "./...", "--allow", allow); code != 0 || !strings.HasSuffix(out, "deslop-please: ok\n") {
		t.Fatalf("code=%d\n%s", code, out)
	}
}

func TestPatternsNarrowTheCheck(t *testing.T) {
	t.Chdir("../../internal/lint/testdata/cycle")
	if code, out := exec(t, "./c/..."); code != 0 {
		t.Fatalf("code=%d\n%s", code, out)
	}
}

func TestUnknownAllowFileIsAnError(t *testing.T) {
	t.Chdir("../../internal/lint/testdata/cycle")
	if code, _ := exec(t, "--allow", "nope"); code != 2 {
		t.Fatalf("code=%d", code)
	}
}
