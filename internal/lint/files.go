package lint

import (
	"fmt"
	"path"

	"github.com/vasilvestre/deslop-please-go/internal/report"
)

// maxFileLines is the longest a source file is before it holds more than one
// responsibility.
const maxFileLines = 800

// bucketNames are package names that say what a package is not about.
var bucketNames = map[string]bool{
	"util": true, "utils": true, "common": true, "helpers": true, "helper": true,
	"misc": true, "shared": true, "base": true, "types": true,
}

// longFiles: a non-test, non-generated file of more than maxFileLines lines.
func longFiles(x *index) []report.Diagnostic {
	var diagnostics []report.Diagnostic
	for _, pkg := range x.model.Packages {
		for _, file := range pkg.Files {
			lines := x.model.Fset.File(file.Pos()).LineCount()
			if lines <= maxFileLines {
				continue
			}
			at := x.where(file.Pos())
			diagnostics = append(diagnostics, report.Diagnostic{
				Lint:    report.LongFile,
				Key:     at.File,
				Message: fmt.Sprintf("`%s` has %d lines, more than %d", at.File, lines, maxFileLines),
				At:      &at,
			})
		}
	}
	return diagnostics
}

// packageBuckets: a package named after a bucket, by the last element of its path.
func packageBuckets(x *index) []report.Diagnostic {
	var diagnostics []report.Diagnostic
	for _, pkg := range x.model.Packages {
		name := path.Base(pkg.Path)
		if pkg.Path == "." {
			name = pkg.Name
		}
		if !bucketNames[name] || len(pkg.Files) == 0 {
			continue
		}
		at := x.where(pkg.Files[0].Package)
		diagnostics = append(diagnostics, report.Diagnostic{
			Lint:    report.PackageBucket,
			Key:     pkg.Path,
			Message: fmt.Sprintf("`%s` is a bucket: its name says nothing about what it owns", pkg.Path),
			At:      &at,
		})
	}
	return diagnostics
}
