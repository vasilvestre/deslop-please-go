package lint

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/vasilvestre/deslop-please-go/internal/report"
)

var syncGuards = map[string]bool{"Mutex": true, "RWMutex": true, "Once": true, "WaitGroup": true}

// exportedMutexes: an exported field of an exported struct that holds a
// sync.Mutex, RWMutex, Once, WaitGroup (value or pointer) or a channel, or an
// embedded sync.Mutex or RWMutex: callers can lock, wait or send themselves.
func exportedMutexes(x *index) []report.Diagnostic {
	var diagnostics []report.Diagnostic
	for _, pkg := range x.model.Packages {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() {
						continue
					}
					if st, ok := ts.Type.(*ast.StructType); ok {
						diagnostics = append(diagnostics, x.exposedFields(pkg.Path, ts.Name.Name, pkg.Info, st)...)
					}
				}
			}
		}
	}
	return diagnostics
}

func (x *index) exposedFields(pkgPath, owner string, info *types.Info, st *ast.StructType) []report.Diagnostic {
	var diagnostics []report.Diagnostic
	for _, field := range st.Fields.List {
		kind := guardKind(info.TypeOf(field.Type), len(field.Names) == 0)
		if kind == "" {
			continue
		}
		names := field.Names
		if len(names) == 0 {
			names = []*ast.Ident{embeddedName(field.Type)}
		}
		for _, name := range names {
			if !name.IsExported() {
				continue
			}
			at := x.where(name.Pos())
			diagnostics = append(diagnostics, report.Diagnostic{
				Lint:    report.ExportedMutex,
				Key:     pkgPath + "." + owner + "." + name.Name,
				Message: fmt.Sprintf("`%s.%s` exposes %s", owner, name.Name, kind),
				At:      &at,
			})
		}
	}
	return diagnostics
}

// guardKind says what t is when callers should not hold it, or "".
func guardKind(t types.Type, embedded bool) string {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if obj.Pkg() != nil && obj.Pkg().Path() == "sync" && syncGuards[obj.Name()] {
			if embedded && obj.Name() != "Mutex" && obj.Name() != "RWMutex" {
				return ""
			}
			return "a sync." + obj.Name()
		}
	}
	if _, ok := t.Underlying().(*types.Chan); ok && !embedded {
		return "a channel"
	}
	return ""
}
