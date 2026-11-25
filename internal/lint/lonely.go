package lint

import (
	"fmt"
	"go/types"

	"github.com/vasilvestre/deslop-please-go/internal/model"
	"github.com/vasilvestre/deslop-please-go/internal/report"
)

// lonelyInterfaces: an exported interface with methods, whose only implementer
// in the analysed packages is a named type of its own package, and that no
// other package uses. Interfaces that embed an interface from outside the
// analysed packages (`error`, `io.Reader`) and constraints are left alone.
func lonelyInterfaces(x *index) []report.Diagnostic {
	usedOutside := map[*thing]bool{}
	x.eachUse(func(u usage) {
		if u.target.pkg != u.pkg {
			usedOutside[u.target] = true
		}
	})
	var diagnostics []report.Diagnostic
	for _, pkg := range x.model.Packages {
		for _, name := range pkg.Types.Scope().Names() {
			obj, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
			if !ok || !obj.Exported() || obj.IsAlias() {
				continue
			}
			iface, ok := obj.Type().Underlying().(*types.Interface)
			if !ok || iface.NumMethods() == 0 || !iface.IsMethodSet() || embedsOutside(x.model, iface) {
				continue
			}
			target := x.byObj[obj]
			if target == nil || usedOutside[target] {
				continue
			}
			implementers := x.implementers(obj, iface)
			if len(implementers) != 1 || implementers[0].Pkg() != obj.Pkg() {
				continue
			}
			at := x.where(obj.Pos())
			diagnostics = append(diagnostics, report.Diagnostic{
				Lint: report.LonelyIface,
				Key:  pkg.Path + "." + obj.Name(),
				Message: fmt.Sprintf("`%s` is an interface with one implementer, `%s`, and no user outside `%s`",
					obj.Name(), implementers[0].Name(), pkg.Path),
				At: &at,
			})
		}
	}
	return diagnostics
}

func (x *index) implementers(self *types.TypeName, iface *types.Interface) []*types.TypeName {
	var found []*types.TypeName
	for _, pkg := range x.model.Packages {
		for _, name := range pkg.Types.Scope().Names() {
			obj, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
			if !ok || obj == self || obj.IsAlias() || types.IsInterface(obj.Type()) || hasTypeParams(obj.Type()) {
				continue
			}
			if types.Implements(obj.Type(), iface) || types.Implements(types.NewPointer(obj.Type()), iface) {
				found = append(found, obj)
			}
		}
	}
	return found
}

func embedsOutside(m *model.Model, iface *types.Interface) bool {
	own := map[*types.Package]bool{}
	for _, pkg := range m.Packages {
		own[pkg.Types] = true
	}
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		if named, ok := iface.EmbeddedType(i).(*types.Named); ok && !own[named.Obj().Pkg()] {
			return true
		}
	}
	return false
}

func hasTypeParams(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.TypeParams().Len() > 0
}
