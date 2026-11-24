package lint

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	"github.com/vsilvestre/deslop-please-go/internal/model"
	"github.com/vsilvestre/deslop-please-go/internal/report"
)

// thing is what a reader learns as one: a type with its fields and methods,
// or a function, variable or constant.
type thing struct {
	pkg    *model.Package
	name   string
	isType bool
	id     int // among its package's things
	at     report.Location
}

// unit is the code of one thing: a declaration, or one spec of a group.
type unit struct {
	node   ast.Node
	thing  *thing
	holder string
	method bool
}

type index struct {
	model  *model.Model
	byObj  map[types.Object]*thing
	things map[*model.Package][]*thing
	units  map[*model.Package][]unit
}

func buildIndex(m *model.Model) *index {
	x := &index{
		model:  m,
		byObj:  map[types.Object]*thing{},
		things: map[*model.Package][]*thing{},
		units:  map[*model.Package][]unit{},
	}
	for _, pkg := range m.Packages {
		x.declare(pkg)
	}
	return x
}

func (x *index) declare(pkg *model.Package) {
	var funcs []*ast.FuncDecl
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				x.declareSpecs(pkg, decl)
			case *ast.FuncDecl:
				funcs = append(funcs, decl)
			}
		}
	}
	for _, fn := range funcs {
		x.declareFunc(pkg, fn)
	}
}

func (x *index) declareSpecs(pkg *model.Package, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch spec := spec.(type) {
		case *ast.TypeSpec:
			t := x.add(pkg, spec.Name, true, spec.Pos())
			x.units[pkg] = append(x.units[pkg], unit{node: spec, thing: t, holder: t.name})
			x.mapMembers(pkg, t, spec.Type)
			x.byObj[pkg.Info.Defs[spec.Name]] = t
		case *ast.ValueSpec:
			t := x.add(pkg, spec.Names[0], false, spec.Pos())
			x.units[pkg] = append(x.units[pkg], unit{node: spec, thing: t, holder: t.name})
			for _, name := range spec.Names {
				x.byObj[pkg.Info.Defs[name]] = t
			}
		}
	}
}

func (x *index) declareFunc(pkg *model.Package, fn *ast.FuncDecl) {
	obj := pkg.Info.Defs[fn.Name]
	owner := x.receiverThing(obj)
	if owner == nil {
		owner = x.add(pkg, fn.Name, false, fn.Pos())
	}
	holder := fn.Name.Name
	if fn.Recv != nil {
		holder = owner.name + "." + holder
	}
	x.byObj[obj] = owner
	x.units[pkg] = append(x.units[pkg], unit{node: fn, thing: owner, holder: holder, method: fn.Recv != nil})
}

func (x *index) receiverThing(obj types.Object) *thing {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Signature().Recv() == nil {
		return nil
	}
	recv := fn.Signature().Recv().Type()
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = ptr.Elem()
	}
	if named, ok := recv.(*types.Named); ok {
		return x.byObj[named.Origin().Obj()]
	}
	return nil
}

// mapMembers makes the fields and interface methods in expr part of t.
func (x *index) mapMembers(pkg *model.Package, t *thing, expr ast.Expr) {
	ast.Inspect(expr, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok {
			return true
		}
		if len(field.Names) == 0 {
			if id := embeddedName(field.Type); id != nil {
				x.byObj[pkg.Info.Defs[id]] = t
			}
		}
		for _, name := range field.Names {
			x.byObj[pkg.Info.Defs[name]] = t
		}
		return true
	})
}

func embeddedName(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return embeddedName(e.X)
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return e.Sel
	case *ast.IndexExpr:
		return embeddedName(e.X)
	}
	return nil
}

func (x *index) add(pkg *model.Package, name *ast.Ident, isType bool, pos token.Pos) *thing {
	t := &thing{pkg: pkg, name: name.Name, isType: isType, id: len(x.things[pkg]), at: x.where(pos)}
	x.things[pkg] = append(x.things[pkg], t)
	return t
}

func (x *index) where(pos token.Pos) report.Location {
	p := x.model.Fset.Position(pos)
	file := p.Filename
	if rel, err := filepath.Rel(x.model.Root, file); err == nil {
		file = rel
	}
	return report.Location{File: filepath.ToSlash(file), Line: p.Line, Col: p.Column}
}

// thingOf is the thing declared by obj, or one of its fields or methods.
func (x *index) thingOf(obj types.Object) *thing {
	switch o := obj.(type) {
	case *types.Var:
		obj = o.Origin()
	case *types.Func:
		obj = o.Origin()
	}
	return x.byObj[obj]
}

// itemName is what of target obj is: the thing itself, or `Type.member`.
func itemName(obj types.Object, target *thing) string {
	if obj.Name() == target.name || obj.Pkg() == nil {
		return target.name
	}
	return target.name + "." + obj.Name()
}

type usage struct {
	id     *ast.Ident
	obj    types.Object
	target *thing
	user   *thing
	holder string
	method bool
	pkg    *model.Package
}

// eachUse calls visit for each identifier of package code that names a thing
// of the analysed packages, in source order, packages in order.
func (x *index) eachUse(visit func(usage)) {
	for _, pkg := range x.model.Packages {
		for _, u := range x.units[pkg] {
			ast.Inspect(u.node, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok {
					return true
				}
				obj := pkg.Info.Uses[id]
				if obj == nil {
					return true
				}
				if target := x.thingOf(obj); target != nil {
					visit(usage{id, obj, target, u.thing, u.holder, u.method, pkg})
				}
				return true
			})
		}
	}
}
