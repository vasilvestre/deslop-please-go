package lint

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/vsilvestre/deslop-please-go/internal/model"
	"github.com/vsilvestre/deslop-please-go/internal/report"
)

// fieldUse is code of another package working on a field.
type fieldUse struct {
	at     report.Location
	pkg    *model.Package
	holder string
	read   bool // a copy of the value: no leak
}

// fieldLeaks: a struct guards its fields when it keeps at least one of them
// to itself and has a method that uses them. Each of its fields that code of
// another package writes, borrows, indexes or calls a method on is then known
// outside it. Reading a value is no leak: whoever reads depends on what it
// is, not on how the struct keeps it. Naming a field in a composite literal
// builds a value: it reads nothing and works on nothing.
func fieldLeaks(x *index) []report.Diagnostic {
	outside := x.outsideFieldUses()
	working := x.workingTypes()
	var diagnostics []report.Diagnostic
	for _, pkg := range x.model.Packages {
		for _, t := range x.things[pkg] {
			if !t.isType || !working[t] {
				continue
			}
			fields := structFields(pkg, t)
			kept := false
			for _, f := range fields {
				kept = kept || len(outside[f]) == 0
			}
			if !kept {
				continue
			}
			for _, f := range fields {
				if leaks(outside[f]) {
					diagnostics = append(diagnostics, leakDiagnostic(t, f, fields, outside))
				}
			}
		}
	}
	return diagnostics
}

func leaks(uses []fieldUse) bool {
	for _, use := range uses {
		if !use.read {
			return true
		}
	}
	return false
}

// structFields are the fields of the struct t declares, markers aside: a
// blank field or an empty struct holds nothing to keep.
func structFields(pkg *model.Package, t *thing) []*types.Var {
	named, ok := pkg.Types.Scope().Lookup(t.name).(*types.TypeName)
	if !ok {
		return nil
	}
	st, ok := named.Type().Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	var fields []*types.Var
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if empty, ok := f.Type().Underlying().(*types.Struct); f.Name() == "_" || ok && empty.NumFields() == 0 {
			continue
		}
		fields = append(fields, f)
	}
	return fields
}

// workingTypes are the types with a method that uses one of their fields.
func (x *index) workingTypes() map[*thing]bool {
	working := map[*thing]bool{}
	x.eachUse(func(u usage) {
		if v, ok := u.obj.(*types.Var); ok && v.IsField() && u.method && u.user == u.target {
			working[u.target] = true
		}
	})
	return working
}

// outsideFieldUses are the uses of each field by code of other packages.
func (x *index) outsideFieldUses() map[*types.Var][]fieldUse {
	outside := map[*types.Var][]fieldUse{}
	for _, pkg := range x.model.Packages {
		for _, u := range x.units[pkg] {
			record := func(f *types.Var, at token.Pos, read bool) {
				f = f.Origin()
				if f.Pkg() == pkg.Types {
					return
				}
				outside[f] = append(outside[f], fieldUse{x.where(at), pkg, u.holder, read})
			}
			walk(u.node, func(n, parent ast.Node) {
				switch n := n.(type) {
				case *ast.SelectorExpr:
					if sel := pkg.Info.Selections[n]; sel != nil && sel.Kind() == types.FieldVal {
						record(sel.Obj().(*types.Var), n.Sel.Pos(), !works(n, parent))
					}
				case *ast.KeyValueExpr:
					if id, ok := n.Key.(*ast.Ident); ok {
						if f, ok := pkg.Info.Uses[id].(*types.Var); ok && f.IsField() {
							record(f, id.Pos(), true)
						}
					}
				}
			})
		}
	}
	return outside
}

// works is whether the context of a field selector writes, borrows, indexes
// or calls into the field, rather than copying it.
func works(sel *ast.SelectorExpr, parent ast.Node) bool {
	switch p := parent.(type) {
	case *ast.AssignStmt:
		for _, lhs := range p.Lhs {
			if lhs == ast.Expr(sel) {
				return true
			}
		}
	case *ast.IncDecStmt:
		return true
	case *ast.UnaryExpr:
		return p.Op == token.AND
	case *ast.IndexExpr:
		return p.X == ast.Expr(sel)
	case *ast.SliceExpr:
		return p.X == ast.Expr(sel)
	case *ast.SelectorExpr:
		return p.X == ast.Expr(sel)
	case *ast.RangeStmt:
		return p.X == ast.Expr(sel)
	case *ast.CallExpr:
		return p.Fun == ast.Expr(sel)
	}
	return false
}

// walk calls visit for each node under root with its parent.
func walk(root ast.Node, visit func(n, parent ast.Node)) {
	var stack []ast.Node
	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		var parent ast.Node
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		visit(n, parent)
		stack = append(stack, n)
		return true
	})
}

func leakDiagnostic(t *thing, field *types.Var, fields []*types.Var, outside map[*types.Var][]fieldUse) report.Diagnostic {
	var uses []fieldUse
	for _, use := range outside[field] {
		if !use.read {
			uses = append(uses, use)
		}
	}
	sort.Slice(uses, func(i, j int) bool { return before(uses[i].at, uses[j].at) })
	places := map[string]int{}
	var notes []string
	for i, use := range uses {
		places[use.pkg.Path]++
		if i < held {
			notes = append(notes, fmt.Sprintf("%s: `%s.%s`", use.at, use.pkg.Path, use.holder))
		}
	}
	if len(uses) > held {
		notes = append(notes, "and "+counted(len(uses)-held, "more place"))
	}
	owner := qualified(t)
	return report.Diagnostic{
		Lint: report.FieldLeak,
		Key:  owner + "." + field.Name(),
		Message: fmt.Sprintf("`%s` is known outside `%s`: %s in %s",
			field.Name(), t.name, counted(len(uses), "place"), counted(len(places), "package")),
		At:    &uses[0].at,
		Notes: notes,
		Picture: report.Spread{
			Owner:    owner,
			Field:    field.Name(),
			Packages: spread(places),
			Others:   otherFields(field, fields, outside),
		},
	}
}

// spread is the packages by places, most first, the tail folded into one.
func spread(places map[string]int) []report.Count {
	var counts []report.Count
	for name, n := range places {
		counts = append(counts, report.Count{N: n, Name: name})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].N != counts[j].N {
			return counts[i].N > counts[j].N
		}
		return counts[i].Name < counts[j].Name
	})
	if len(counts) > held {
		more, n := counts[held:], 0
		for _, c := range more {
			n += c.N
		}
		counts = append(counts[:held], report.Count{N: n, Name: "in " + counted(len(more), "more package")})
	}
	return counts
}

// otherFields says of the other fields which are known outside too, and which
// the struct keeps.
func otherFields(field *types.Var, fields []*types.Var, outside map[*types.Var][]fieldUse) []string {
	var known, kept []string
	for _, f := range fields {
		switch {
		case f == field:
		case len(outside[f]) == 0:
			kept = append(kept, f.Name())
		case leaks(outside[f]):
			known = append(known, f.Name())
		}
	}
	var others []string
	if len(known) > 0 {
		others = append(others, "also known outside: "+listed(known))
	}
	if len(kept) > 0 {
		others = append(others, "kept: "+listed(kept))
	}
	return others
}
