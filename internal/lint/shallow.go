package lint

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/vsilvestre/deslop-please-go/internal/model"
	"github.com/vsilvestre/deslop-please-go/internal/report"
)

// breakEven is what a package hides for each function it shows. Its interface
// is the cost of using it and what it hides the benefit, so it is shallow
// when it shows at least as much as it hides.
const breakEven = 1

// protocolMethods are the methods that satisfy an interface of a library:
// callers reach them through it, so they are no part of a package's own API.
var protocolMethods = map[string]bool{
	"String": true, "Error": true, "Len": true, "Less": true, "Swap": true,
	"ServeHTTP": true, "MarshalJSON": true, "UnmarshalJSON": true,
	"MarshalText": true, "UnmarshalText": true, "Unwrap": true,
	"Close": true, "Read": true, "Write": true,
}

// function is a declared function or method of a package.
type function struct {
	obj     *types.Func
	decl    *ast.FuncDecl
	home    *thing
	wrapper bool
	target  *types.Func // what a wrapper calls
}

func (f *function) name() string {
	if f.decl.Recv != nil {
		return f.home.name + "." + f.decl.Name.Name
	}
	return f.decl.Name.Name
}

// working is whether a function counts: wrappers and protocol methods do not.
func (f *function) working() bool {
	return !f.wrapper && !(f.decl.Recv != nil && protocolMethods[f.decl.Name.Name]) &&
		f.decl.Name.Name != "init" && f.decl.Body != nil
}

// shallowModules: a package that shows more functions than a reader holds in
// mind, at least as many as it hides, and a list that long: more things, or
// more functions of one thing. A thing's functions are its methods and the
// package's free functions that use it; one that uses none of the package's
// types is a thing of its own. A package shows the working functions that
// code of other packages uses, through wrappers.
func shallowModules(x *index) []report.Diagnostic {
	funcs := x.functions()
	shown := map[*model.Package]map[*function]bool{}
	known := map[*thing]map[*thing]bool{}
	x.eachUse(func(u usage) {
		if u.target.isType && u.target.pkg == u.pkg {
			add(known, u.user, u.target)
		}
		fn := funcs[funcOf(u.obj)]
		if fn == nil || u.pkg == fn.home.pkg {
			return
		}
		if fn.wrapper {
			fn = funcs[fn.target]
		}
		if fn != nil && fn.working() {
			if shown[fn.home.pkg] == nil {
				shown[fn.home.pkg] = map[*function]bool{}
			}
			shown[fn.home.pkg][fn] = true
		}
	})
	total := map[*model.Package]int{}
	for _, fn := range funcs {
		if fn.working() {
			total[fn.home.pkg]++
		}
	}
	var diagnostics []report.Diagnostic
	for _, pkg := range x.model.Packages {
		shown := shown[pkg]
		hidden := total[pkg] - len(shown)
		if pkg.Name == "main" || len(shown) <= held || hidden > len(shown)*breakEven {
			continue
		}
		notes, long := tooLong(lists(shown, known))
		if len(notes) == 0 {
			continue
		}
		at := x.firstFile(pkg)
		diagnostics = append(diagnostics, report.Diagnostic{
			Lint: report.ShallowModule,
			Key:  pkg.Path,
			Message: fmt.Sprintf("`%s` is shallow: it shows %d of its %s",
				pkg.Path, len(shown), counted(total[pkg], "function")),
			At:      &at,
			Notes:   notes,
			Picture: report.Lists{Shown: len(shown), Hidden: hidden, Long: long},
		})
	}
	return diagnostics
}

func add(sets map[*thing]map[*thing]bool, key, value *thing) {
	if sets[key] == nil {
		sets[key] = map[*thing]bool{}
	}
	sets[key][value] = true
}

func funcOf(obj types.Object) *types.Func {
	if fn, ok := obj.(*types.Func); ok {
		return fn.Origin()
	}
	return nil
}

// functions are the declared functions of the analysed packages.
func (x *index) functions() map[*types.Func]*function {
	funcs := map[*types.Func]*function{}
	for _, pkg := range x.model.Packages {
		for _, u := range x.units[pkg] {
			decl, ok := u.node.(*ast.FuncDecl)
			if !ok {
				continue
			}
			obj, _ := pkg.Info.Defs[decl.Name].(*types.Func)
			if obj == nil {
				continue
			}
			fn := &function{obj: obj, decl: decl, home: u.thing}
			fn.wrapper, fn.target = forwards(pkg, decl)
			funcs[obj] = fn
		}
	}
	return funcs
}

func (x *index) firstFile(pkg *model.Package) report.Location {
	first := report.Location{Line: 1, Col: 1}
	for _, file := range pkg.Files {
		at := x.where(file.Pos())
		if first.File == "" || at.File < first.File {
			first = report.Location{File: at.File, Line: 1, Col: 1}
		}
	}
	return first
}

// forwards is whether decl only hands its work on: a body of one statement
// that holds no more than selectors, literals and one call of simple
// arguments. The function it calls, if it is a declared one, is its target.
func forwards(pkg *model.Package, decl *ast.FuncDecl) (bool, *types.Func) {
	if decl.Body == nil || len(decl.Body.List) != 1 {
		return false, nil
	}
	var expr ast.Expr
	switch s := decl.Body.List[0].(type) {
	case *ast.ReturnStmt:
		if len(s.Results) != 1 {
			return false, nil
		}
		expr = s.Results[0]
	case *ast.ExprStmt:
		expr = s.X
	default:
		return false, nil
	}
	var target *types.Func
	if call, ok := expr.(*ast.CallExpr); ok {
		target = calledFunc(pkg, call)
	}
	return simple(expr), target
}

func calledFunc(pkg *model.Package, call *ast.CallExpr) *types.Func {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return funcOf(pkg.Info.Uses[fun])
	case *ast.SelectorExpr:
		return funcOf(pkg.Info.Uses[fun.Sel])
	}
	return nil
}

func simple(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.SelectorExpr:
		return simple(e.X)
	case *ast.StarExpr:
		return simple(e.X)
	case *ast.ParenExpr:
		return simple(e.X)
	case *ast.UnaryExpr:
		return simple(e.X)
	case *ast.IndexExpr:
		return simple(e.X) && simple(e.Index)
	case *ast.CallExpr:
		for _, arg := range e.Args {
			if !simple(arg) {
				return false
			}
		}
		return simple(e.Fun)
	}
	return false
}

type list struct {
	thing     string
	functions []string
}

// lists are the lists a reader learns what a package shows by, one at a time:
// each thing shown, with the names of its functions there, widest first. A
// method is on its type's list, a free function on that of each type of its
// package it uses, or on its own if it uses none.
func lists(shown map[*function]bool, known map[*thing]map[*thing]bool) []list {
	byThing := map[*thing][]string{}
	for fn := range shown {
		on := []*thing{fn.home}
		if fn.decl.Recv == nil {
			var used []*thing
			for t := range known[fn.home] {
				used = append(used, t)
			}
			if len(used) > 0 {
				on = used
			}
		}
		for _, t := range on {
			byThing[t] = append(byThing[t], fn.name())
		}
	}
	var result []list
	for t, names := range byThing {
		sort.Strings(names)
		result = append(result, list{t.name, names})
	}
	sort.Slice(result, func(i, j int) bool {
		if len(result[i].functions) != len(result[j].functions) {
			return len(result[i].functions) > len(result[j].functions)
		}
		return result[i].thing < result[j].thing
	})
	return result
}

// tooLong notes each list longer than a reader holds in mind, the list of
// things first, then each thing's functions.
func tooLong(lists []list) ([]string, []report.Count) {
	var notes []string
	var long []report.Count
	if len(lists) > held {
		names := make([]string, len(lists))
		for i, l := range lists {
			names[i] = l.thing
		}
		notes = append(notes, fmt.Sprintf("shows %s: %s", counted(len(names), "thing"), listed(names)))
		long = append(long, report.Count{N: len(names), Name: "things"})
	}
	for _, l := range lists {
		if len(l.functions) > held {
			notes = append(notes, fmt.Sprintf("`%s` shows %s: %s", l.thing, counted(len(l.functions), "function"), listed(l.functions)))
			long = append(long, report.Count{N: len(l.functions), Name: l.thing})
		}
	}
	return notes, long
}
