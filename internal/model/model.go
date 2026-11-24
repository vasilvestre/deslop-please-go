// Package model loads the Go packages to lint, type-checked.
package model

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Package struct {
	Path  string // relative to its module, "." for the module's root
	Name  string
	Types *types.Package
	Info  *types.Info
	Files []*ast.File
}

type Model struct {
	Root     string // directory file names are shown relative to
	Fset     *token.FileSet
	Packages []*Package
}

const mode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
	packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
	packages.NeedImports | packages.NeedDeps | packages.NeedModule

// Load type-checks the packages of the main module that match patterns,
// without their tests and without generated files.
func Load(dir string, patterns []string) (*Model, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	cfg := &packages.Config{Mode: mode, Dir: root}
	loaded, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	model := &Model{Root: root}
	for _, p := range loaded {
		if len(p.Errors) > 0 {
			return nil, fmt.Errorf("%s: %v", p.PkgPath, p.Errors[0])
		}
		if p.Module == nil || !p.Module.Main {
			continue
		}
		model.Fset = p.Fset
		model.Packages = append(model.Packages, &Package{
			Path:  relative(p.PkgPath, p.Module.Path),
			Name:  p.Name,
			Types: p.Types,
			Info:  p.TypesInfo,
			Files: sourceFiles(p.Syntax),
		})
	}
	sort.Slice(model.Packages, func(i, j int) bool { return model.Packages[i].Path < model.Packages[j].Path })
	return model, nil
}

func relative(path, module string) string {
	rel := strings.Trim(strings.TrimPrefix(path, module), "/")
	if rel == "" {
		return "."
	}
	return rel
}

func sourceFiles(files []*ast.File) []*ast.File {
	var kept []*ast.File
	for _, file := range files {
		if !ast.IsGenerated(file) {
			kept = append(kept, file)
		}
	}
	return kept
}
